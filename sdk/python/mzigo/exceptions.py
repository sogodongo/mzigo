"""
Mzigo SDK exception hierarchy.

All exceptions are importable from the top-level package:
    from mzigo import ContractViolationError, GatewayError

We use a hierarchy rather than a single exception class so callers
can catch at the right level of granularity:

    try:
        producer.produce(topic, payload)
    except ContractViolationError as e:
        # Schema problem with the payload. Fix the producer.
        log.error("contract violation", fields=e.violations)
    except GatewayError as e:
        # Infrastructure problem. Retry or alert.
        log.error("gateway unreachable", error=e)
    except MzigoError:
        # Catch-all for any SDK error.
        raise
"""

from __future__ import annotations

from dataclasses import dataclass, field


class MzigoError(Exception):
    """Base class for all Mzigo SDK exceptions."""


class GatewayError(MzigoError):
    """
    The gateway returned an unexpected error or was unreachable.
    This is an infrastructure problem, not a data problem.
    Appropriate response: retry with backoff, alert on-call if sustained.
    """


class GatewayTimeoutError(GatewayError):
    """The gateway did not respond within the configured timeout."""


class ContractNotFoundError(MzigoError):
    """
    No active contract exists for the requested topic.
    The producer is attempting to write to a topic that has not been
    registered in the contract registry.
    """

    def __init__(self, topic: str) -> None:
        self.topic = topic
        super().__init__(f"no active contract found for topic {topic!r}")


@dataclass
class Violation:
    """A single contract violation on a specific field."""
    type: str
    field: str | None
    message: str


class ContractViolationError(MzigoError):
    """
    The message was rejected by the gateway because it violates
    the active contract for the topic.

    This is a producer-side problem: the payload does not conform
    to what the contract requires. Fix the producer, not the infrastructure.

    violations contains the structured detail from the gateway response
    so the producer can surface the exact fields that need correction.
    """

    def __init__(self, topic: str, violations: list[Violation]) -> None:
        self.topic = topic
        self.violations = violations
        detail = "; ".join(
            f"{v.field or 'message'}: {v.message}" for v in violations
        )
        super().__init__(
            f"contract violation on topic {topic!r}: {detail}"
        )

    def first_violation(self) -> Violation | None:
        return self.violations[0] if self.violations else None


class ContractVersionMismatchWarning(UserWarning):
    """
    The producer declared a contract version that differs from the
    active version in the registry. The message was still accepted
    (the gateway enforces the active version regardless of what the
    producer declares), but the producer should update its contract reference.
    """
