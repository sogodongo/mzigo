"""
Kafka consumer for the lineage worker.

Consumer group design: each lineage worker instance is assigned a subset
of partitions by Kafka's group coordinator. We use manual offset commit
after successful processing to ensure at-least-once delivery semantics.
A worker crash before commit means messages are reprocessed on restart.
Lineage event emission is idempotent at the Marquez level (same run_id
for the same offset would overwrite), so reprocessing is safe.

We consume from two topics:
- mzigo.gateway.audit: every message the gateway accepted, with producer metadata
- mzigo.gateway.violations: every message the gateway rejected

This gives us lineage for both the happy path and the failure path.
"""

from __future__ import annotations

import signal
import time
from types import FrameType

import structlog
from confluent_kafka import Consumer, KafkaError, KafkaException, Message

from config import Settings
from lineage.edge_store import EdgeStore
from lineage.emitter import LineageEmitter
from lineage.extractor import FieldExtractor
from metrics import (
    consumer_lag,
    messages_failed,
    messages_processed,
    processing_duration,
)

log = structlog.get_logger(__name__)


class LineageConsumer:
    def __init__(
        self,
        settings: Settings,
        emitter: LineageEmitter,
        extractor: FieldExtractor,
        edge_store: EdgeStore,
    ) -> None:
        self._settings = settings
        self._emitter = emitter
        self._extractor = extractor
        self._edge_store = edge_store
        self._running = False
        self._messages_since_commit = 0

        self._consumer = Consumer({
            "bootstrap.servers": settings.kafka_bootstrap_servers,
            "group.id": settings.kafka_consumer_group,
            "auto.offset.reset": "earliest",
            # We commit manually after processing to guarantee at-least-once.
            # auto.commit would commit offsets on poll regardless of whether
            # processing succeeded, which risks silently losing lineage events.
            "enable.auto.commit": False,
            "session.timeout.ms": 30000,
            "heartbeat.interval.ms": 10000,
        })

    def run(self) -> None:
        self._running = True
        self._register_signal_handlers()

        topics = [*self._settings.kafka_topics, "mzigo.gateway.audit", "mzigo.gateway.violations"]
        self._consumer.subscribe(topics, on_assign=self._on_assign, on_revoke=self._on_revoke)

        log.info("lineage_consumer_started", topics=topics)

        try:
            while self._running:
                msg = self._consumer.poll(timeout=self._settings.kafka_poll_timeout_seconds)

                if msg is None:
                    continue

                if msg.error():
                    self._handle_consumer_error(msg.error())
                    continue

                self._process_message(msg)

        finally:
            # Commit any pending offsets before shutdown to minimize reprocessing.
            self._commit()
            self._consumer.close()
            self._emitter.close()
            log.info("lineage_consumer_stopped")

    def _process_message(self, msg: Message) -> None:
        topic = msg.topic()
        start = time.monotonic()

        try:
            if topic == "mzigo.gateway.violations":
                self._handle_violation(msg)
            else:
                self._handle_audit(msg)

            self._messages_since_commit += 1
            messages_processed.labels(topic=topic).inc()

            if self._messages_since_commit >= self._settings.kafka_commit_interval:
                self._commit()

        except Exception as exc:
            messages_failed.labels(topic=topic, reason=type(exc).__name__).inc()
            log.error(
                "message_processing_failed",
                topic=topic,
                partition=msg.partition(),
                offset=msg.offset(),
                error=str(exc),
            )
        finally:
            processing_duration.observe(time.monotonic() - start)

    def _handle_audit(self, msg: Message) -> None:
        """Process an accepted message event from the gateway audit topic."""
        metadata = self._extractor.extract(
            topic=msg.topic(),
            payload=msg.value(),
        )

        producer_id = self._get_header(msg, "x-producer-id") or "unknown"
        contract_id = self._get_header(msg, "x-contract-id") or ""

        self._emitter.emit_complete(
            metadata=metadata,
            producer_id=producer_id,
            partition=msg.partition(),
            offset=msg.offset(),
        )

        log.debug(
            "lineage_emitted",
            topic=msg.topic(),
            producer_id=producer_id,
            fields=len(metadata.fields),
        )

    def _handle_violation(self, msg: Message) -> None:
        """Process a rejection event from the gateway violations topic."""
        import json
        try:
            body = json.loads(msg.value())
        except (json.JSONDecodeError, TypeError):
            return

        self._emitter.emit_violation(
            topic=body.get("topic", "unknown"),
            producer_id=body.get("producer_id", "unknown"),
            violation_type=body.get("violation_type", "UNKNOWN"),
        )

    def _commit(self) -> None:
        try:
            self._consumer.commit(asynchronous=False)
            self._messages_since_commit = 0
        except KafkaException as exc:
            log.error("offset_commit_failed", error=str(exc))

    def _on_assign(self, consumer, partitions) -> None:
        log.info("partitions_assigned", count=len(partitions))

    def _on_revoke(self, consumer, partitions) -> None:
        # Commit before rebalance to avoid reprocessing messages
        # that were already emitted to Marquez in this session.
        self._commit()
        log.info("partitions_revoked", count=len(partitions))

    def _handle_consumer_error(self, error: KafkaError) -> None:
        if error.code() == KafkaError._PARTITION_EOF:
            # Normal end-of-partition signal. Not an error.
            return
        log.error("kafka_consumer_error", code=error.code(), message=error.str())

    def _get_header(self, msg: Message, key: str) -> str | None:
        headers = msg.headers() or []
        for k, v in headers:
            if k == key and v:
                return v.decode("utf-8", errors="replace")
        return None

    def _register_signal_handlers(self) -> None:
        def _stop(sig: int, _frame: FrameType | None) -> None:
            log.info("shutdown_signal_received", signal=sig)
            self._running = False

        signal.signal(signal.SIGTERM, _stop)
        signal.signal(signal.SIGINT, _stop)
