"""
MzigoProducer: the primary interface for contract-compliant message production.

Design goals:
- One import, one class, two methods (produce, close)
- ContractRef is explicit: producers know which contract they are using
- Exceptions are typed and actionable
- Context manager support for clean resource lifecycle

Usage:

    from mzigo import MzigoProducer, ContractRef, ProducerConfig

    config = ProducerConfig(
        gateway_url="https://mzigo-gateway.internal",
        producer_id="payments-service",
    )
    contract = ContractRef(
        topic="payments.transactions",
        contract_id="a1b2c3d4",
        version="2.1.0",
    )

    with MzigoProducer(config) as producer:
        result = producer.produce(
            contract=contract,
            payload={"transaction_id": "...", "amount_cents": 1000},
            key="txn-abc",
        )
        print(result.message_id)
"""

from __future__ import annotations

from typing import Any

from mzigo._http import Transport
from mzigo.exceptions import ContractViolationError
from mzigo.types import ContractRef, ProduceResult, ProducerConfig


class MzigoProducer:
    """
    Synchronous contract-aware Kafka producer.

    Thread safety: MzigoProducer is safe to use from multiple threads.
    The underlying httpx.Client manages a connection pool per instance.
    For high-throughput multi-threaded workloads, create one producer
    per thread rather than sharing a single instance to avoid pool contention.
    """

    def __init__(self, config: ProducerConfig) -> None:
        self._config = config
        self._transport = Transport(config)

    def produce(
        self,
        contract: ContractRef,
        payload: dict[str, Any],
        key: str | None = None,
    ) -> ProduceResult:
        """
        Produce a message to the gateway.

        Raises:
            ContractViolationError: The payload violates the active contract.
                                    Fix the payload; retrying unchanged will fail.
            ContractNotFoundError:  No active contract for the topic.
                                    Register a contract before producing.
            GatewayTimeoutError:    Gateway did not respond in time. Retry.
            GatewayError:           Transient gateway error. Retry with backoff.
        """
        return self._transport.produce(
            topic=contract.topic,
            contract_id=contract.contract_id,
            contract_version=contract.version,
            key=key,
            payload=payload,
        )

    def close(self) -> None:
        self._transport.close()

    def __enter__(self) -> "MzigoProducer":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()
