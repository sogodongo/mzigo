from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass(frozen=True)
class ContractRef:
    """
    A reference to a specific contract version.
    Producers pin this to ensure they know which contract version their
    code was written against. The gateway validates against the active
    version; the declared version is used for mismatch detection.
    """
    topic: str
    contract_id: str
    version: str


@dataclass(frozen=True)
class ProduceResult:
    """
    Returned by MzigoProducer.produce() on success.
    message_id is the Kafka offset string: "{topic}:{partition}:{offset}".
    Use it for correlation in audit logs.
    """
    message_id: str
    topic: str
    duration_ms: int


@dataclass
class ProducerConfig:
    """
    Configuration for MzigoProducer.

    gateway_url: Base URL of the Mzigo gateway service.
    producer_id: Identifies this producer in violation logs and lineage events.
                 Use a stable, descriptive name: "payments-service", not "worker-1".
    timeout:     Per-request timeout in seconds. Should be well above your p99
                 gateway latency budget. 5s is a safe default for most deployments.
    max_retries: Retries on transient gateway errors (5xx, timeouts).
                 Contract violations (422) are not retried: retrying a bad payload
                 will always fail.
    """
    gateway_url: str
    producer_id: str
    timeout: float = 5.0
    max_retries: int = 3
    retry_wait_seconds: float = 0.5
