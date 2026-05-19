"""
Mzigo Python SDK

Produces contract-compliant messages through the Mzigo gateway.

Quickstart:

    from mzigo import MzigoProducer, ContractRef, ProducerConfig

    config = ProducerConfig(
        gateway_url="https://mzigo-gateway.internal",
        producer_id="my-service",
    )

    contract = ContractRef(
        topic="payments.transactions",
        contract_id="your-contract-id",
        version="2.1.0",
    )

    with MzigoProducer(config) as producer:
        result = producer.produce(
            contract=contract,
            payload={
                "transaction_id": "txn-abc123",
                "amount_cents": 5000,
                "currency": "USD",
            },
        )

For async producers:

    from mzigo import AsyncMzigoProducer

    async with AsyncMzigoProducer(config) as producer:
        result = await producer.produce(contract=contract, payload={...})

Error handling:

    from mzigo import ContractViolationError, GatewayError

    try:
        producer.produce(contract, payload)
    except ContractViolationError as e:
        # Fix the payload
        for v in e.violations:
            print(v.field, v.message)
    except GatewayError:
        # Retry or alert
        raise
"""

from mzigo.async_producer import AsyncMzigoProducer
from mzigo.exceptions import (
    ContractNotFoundError,
    ContractViolationError,
    ContractVersionMismatchWarning,
    GatewayError,
    GatewayTimeoutError,
    MzigoError,
    Violation,
)
from mzigo.producer import MzigoProducer
from mzigo.types import ContractRef, ProduceResult, ProducerConfig

__all__ = [
    "MzigoProducer",
    "AsyncMzigoProducer",
    "ProducerConfig",
    "ContractRef",
    "ProduceResult",
    "Violation",
    "MzigoError",
    "GatewayError",
    "GatewayTimeoutError",
    "ContractViolationError",
    "ContractNotFoundError",
    "ContractVersionMismatchWarning",
]

__version__ = "0.1.0"
