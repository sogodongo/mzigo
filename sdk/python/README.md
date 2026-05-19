# Mzigo Python SDK

Producer SDK for the Mzigo streaming data contracts platform.

Mzigo enforces data contracts at the gateway before messages reach Kafka.
This SDK handles the gateway protocol so your producer code does not have to.

## Installation

```bash
pip install mzigo
```

## Quickstart

```python
from mzigo import MzigoProducer, ContractRef, ProducerConfig

config = ProducerConfig(
    gateway_url="https://mzigo-gateway.your-cluster.internal",
    producer_id="payments-service",
)

contract = ContractRef(
    topic="payments.transactions",
    contract_id="your-contract-id",   # from the Mzigo catalog
    version="2.1.0",
)

with MzigoProducer(config) as producer:
    result = producer.produce(
        contract=contract,
        payload={
            "transaction_id": "txn-abc123",
            "amount_cents": 5000,
            "currency": "USD",
            "status": "AUTHORIZED",
            "produced_at": 1706000000000,
        },
        key="txn-abc123",
    )
    print(result.message_id)  # payments.transactions:0:42
```

## Error Handling

```python
from mzigo import ContractViolationError, GatewayError, ContractNotFoundError

try:
    result = producer.produce(contract, payload)

except ContractViolationError as e:
    # The payload violates the contract. Fix the payload.
    # Retrying the same payload will fail again.
    for violation in e.violations:
        print(f"field {violation.field!r}: {violation.message}")

except ContractNotFoundError as e:
    # No active contract is registered for this topic.
    # Register a contract in the Mzigo catalog before producing.
    print(f"no contract for topic: {e.topic}")

except GatewayError as e:
    # Transient infrastructure problem. The SDK retries automatically,
    # so this exception means all retries were exhausted.
    raise
```

## Async Usage

```python
from mzigo import AsyncMzigoProducer

async with AsyncMzigoProducer(config) as producer:
    result = await producer.produce(contract=contract, payload=payload)
```

## Configuration

```python
from mzigo import ProducerConfig

config = ProducerConfig(
    gateway_url="https://...",     # Required
    producer_id="my-service",      # Required. Use a stable service name.
    timeout=5.0,                   # Per-request timeout in seconds
    max_retries=3,                 # Retries on transient gateway errors
    retry_wait_seconds=0.5,        # Wait between retries
)
```

## Finding Your Contract ID

Open the Mzigo catalog and navigate to your topic's contract page.
The contract ID is shown in the contract header.

Alternatively, your platform team can configure the `ContractResolver`
to look up IDs automatically:

```python
from mzigo.contract import ContractResolver

resolver = ContractResolver(contracts_url="https://mzigo-contracts.internal")
contract_ref = resolver.resolve("payments.transactions")
```

## What Happens When a Message Is Rejected

The gateway returns a structured rejection with the specific fields
that violated the contract. The SDK maps this to a `ContractViolationError`
with a typed `violations` list. Your error handler gets field names and
human-readable messages, not an HTTP response body to parse.

The gateway does not retry rejections. A rejected message has a data
problem, not a transient infrastructure problem.
