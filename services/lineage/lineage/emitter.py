"""
OpenLineage event emitter.

Constructs RunEvents from extracted message metadata and emits them
to the configured Marquez endpoint.

OpenLineage concepts used here:
- Run: a single execution of a job (one message batch in our case)
- Job: the producer that wrote the data (identified by producer_id)
- Dataset: the Kafka topic as input, the downstream dataset as output
- Facets: structured metadata attached to runs, jobs, and datasets

We emit two event types:
- START: when we begin processing a batch (optional, can be skipped for
  low-volume topics where the overhead isn't worth the granularity)
- COMPLETE: after successful processing, with field-level schema facets

Column-level lineage is attached as a ColumnLineageDatasetFacet on the
output dataset. This is what powers the field-level lineage graph in Marquez.
"""

from __future__ import annotations

import uuid
from datetime import datetime, timezone
from typing import Any

import httpx
import structlog

from lineage.extractor import MessageMetadata
from metrics import lineage_emit_failures, lineage_events_emitted

log = structlog.get_logger(__name__)


class LineageEmitter:
    def __init__(self, marquez_url: str, namespace: str, emit_timeout: float) -> None:
        self._marquez_url = marquez_url.rstrip("/")
        self._namespace = namespace
        self._client = httpx.Client(timeout=emit_timeout)

    def emit_complete(self, metadata: MessageMetadata, producer_id: str, partition: int, offset: int) -> None:
        """
        Emit a COMPLETE RunEvent for a successfully processed message.
        This is the primary lineage signal: it records that data moved
        from a producer job to a Kafka topic dataset.
        """
        run_id = str(uuid.uuid4())
        now = datetime.now(tz=timezone.utc).isoformat()

        event = {
            "eventType": "COMPLETE",
            "eventTime": now,
            "run": {
                "runId": run_id,
                "facets": {
                    "kafkaOffset": {
                        "_producer": "https://mzigo.io",
                        "_schemaURL": "https://mzigo.io/facets/kafka-offset/v1",
                        "topic": metadata.topic,
                        "partition": partition,
                        "offset": offset,
                    }
                },
            },
            "job": {
                "namespace": self._namespace,
                "name": producer_id,
                "facets": {
                    "documentation": {
                        "_producer": "https://mzigo.io",
                        "_schemaURL": "https://openlineage.io/spec/facets/1-0-0/DocumentationJobFacet.json",
                        "description": f"Producer writing to {metadata.topic}",
                    }
                },
            },
            "inputs": [],
            "outputs": [self._build_output_dataset(metadata)],
        }

        self._send(event)

    def emit_violation(self, topic: str, producer_id: str, violation_type: str) -> None:
        """
        Emit a FAIL RunEvent when the gateway rejects a message.
        This records contract violations in the lineage graph so
        operators can see violation frequency alongside data flow.
        """
        run_id = str(uuid.uuid4())
        now = datetime.now(tz=timezone.utc).isoformat()

        event = {
            "eventType": "FAIL",
            "eventTime": now,
            "run": {
                "runId": run_id,
                "facets": {
                    "errorMessage": {
                        "_producer": "https://mzigo.io",
                        "_schemaURL": "https://openlineage.io/spec/facets/1-0-0/ErrorMessageRunFacet.json",
                        "message": f"Contract violation: {violation_type}",
                        "programmingLanguage": "N/A",
                    }
                },
            },
            "job": {
                "namespace": self._namespace,
                "name": producer_id,
                "facets": {},
            },
            "inputs": [],
            "outputs": [{"namespace": self._namespace, "name": topic, "facets": {}}],
        }

        self._send(event)

    def _build_output_dataset(self, metadata: MessageMetadata) -> dict[str, Any]:
        dataset: dict[str, Any] = {
            "namespace": self._namespace,
            "name": metadata.topic,
            "facets": {},
        }

        if metadata.fields:
            dataset["facets"]["schema"] = self._build_schema_facet(metadata)
            dataset["facets"]["columnLineage"] = self._build_column_lineage_facet(metadata)

        return dataset

    def _build_schema_facet(self, metadata: MessageMetadata) -> dict[str, Any]:
        return {
            "_producer": "https://mzigo.io",
            "_schemaURL": "https://openlineage.io/spec/facets/1-0-0/SchemaDatasetFacet.json",
            "fields": [
                {
                    "name": f.path,
                    "type": f.value_type,
                }
                for f in metadata.fields
            ],
        }

    def _build_column_lineage_facet(self, metadata: MessageMetadata) -> dict[str, Any]:
        # Column-level lineage maps each output field back to its input.
        # For producer-to-topic lineage the fields flow directly: the producer
        # wrote field X, therefore the topic dataset has field X from the producer.
        # This becomes more interesting when we track Flink job transformations.
        fields: dict[str, Any] = {}
        for f in metadata.fields:
            fields[f.path] = {
                "inputFields": [
                    {
                        "namespace": self._namespace,
                        "dataset": f"producer.{metadata.topic}",
                        "field": f.path,
                    }
                ],
                "transformationDescription": "direct",
                "transformationType": "IDENTITY",
            }

        return {
            "_producer": "https://mzigo.io",
            "_schemaURL": "https://openlineage.io/spec/facets/1-0-0/ColumnLineageDatasetFacet.json",
            "fields": fields,
        }

    def _send(self, event: dict[str, Any]) -> None:
        event_type = event.get("eventType", "UNKNOWN")
        try:
            response = self._client.post(
                f"{self._marquez_url}/api/v1/lineage",
                json=event,
            )
            response.raise_for_status()
            lineage_events_emitted.labels(event_type=event_type).inc()
        except httpx.TimeoutException:
            lineage_emit_failures.inc()
            log.warning("lineage_emit_timeout", event_type=event_type)
        except httpx.HTTPStatusError as exc:
            lineage_emit_failures.inc()
            log.error(
                "lineage_emit_http_error",
                event_type=event_type,
                status_code=exc.response.status_code,
            )
        except Exception as exc:
            lineage_emit_failures.inc()
            log.error("lineage_emit_error", event_type=event_type, error=str(exc))

    def close(self) -> None:
        self._client.close()
