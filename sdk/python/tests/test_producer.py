"""
Producer tests mock the gateway at the HTTP level using respx.
This tests the full SDK path from produce() call to exception, without
requiring a live gateway. The transport layer is not bypassed because
that is where the response parsing and exception mapping live.
"""

import pytest
import respx
import httpx

from mzigo import (
    MzigoProducer,
    AsyncMzigoProducer,
    ContractRef,
    ProducerConfig,
    ContractViolationError,
    ContractNotFoundError,
    GatewayError,
    GatewayTimeoutError,
)


GATEWAY_URL = "http://gateway.test"

CONTRACT = ContractRef(
    topic="payments.transactions",
    contract_id="contract-abc",
    version="2.1.0",
)

PAYLOAD = {
    "transaction_id": "txn-001",
    "amount_cents": 5000,
    "currency": "USD",
}


@pytest.fixture
def config() -> ProducerConfig:
    return ProducerConfig(
        gateway_url=GATEWAY_URL,
        producer_id="test-service",
        timeout=2.0,
        max_retries=2,
        retry_wait_seconds=0.0,
    )


@pytest.fixture
def producer(config) -> MzigoProducer:
    p = MzigoProducer(config)
    yield p
    p.close()


# Happy path

@respx.mock
def test_produce_accepted_returns_result(producer):
    respx.post(f"{GATEWAY_URL}/v1/produce").mock(
        return_value=httpx.Response(202, json={
            "status": "ACCEPTED",
            "message_id": "payments.transactions:0:42",
            "duration_ms": 3,
        })
    )

    result = producer.produce(CONTRACT, PAYLOAD)

    assert result.message_id == "payments.transactions:0:42"
    assert result.duration_ms == 3


# Contract violations

@respx.mock
def test_produce_contract_violation_raises_typed_error(producer):
    respx.post(f"{GATEWAY_URL}/v1/produce").mock(
        return_value=httpx.Response(422, json={
            "status": "REJECTED",
            "topic": "payments.transactions",
            "violations": [
                {
                    "type": "MISSING_REQUIRED_FIELD",
                    "field": "transaction_id",
                    "message": "required field 'transaction_id' is absent",
                }
            ],
        })
    )

    with pytest.raises(ContractViolationError) as exc_info:
        producer.produce(CONTRACT, {})

    err = exc_info.value
    assert err.topic == "payments.transactions"
    assert len(err.violations) == 1
    assert err.violations[0].field == "transaction_id"
    assert err.violations[0].type == "MISSING_REQUIRED_FIELD"


@respx.mock
def test_produce_no_contract_raises_not_found(producer):
    respx.post(f"{GATEWAY_URL}/v1/produce").mock(
        return_value=httpx.Response(422, json={
            "status": "REJECTED",
            "topic": "payments.transactions",
            "violations": [
                {
                    "type": "NO_CONTRACT_FOR_TOPIC",
                    "message": "no active contract found for topic",
                }
            ],
        })
    )

    with pytest.raises(ContractNotFoundError) as exc_info:
        producer.produce(CONTRACT, PAYLOAD)

    assert exc_info.value.topic == "payments.transactions"


# Gateway errors and retries

@respx.mock
def test_produce_gateway_500_raises_gateway_error(producer):
    respx.post(f"{GATEWAY_URL}/v1/produce").mock(
        return_value=httpx.Response(500)
    )

    with pytest.raises(GatewayError):
        producer.produce(CONTRACT, PAYLOAD)


@respx.mock
def test_produce_retries_on_gateway_error(config):
    call_count = 0

    def side_effect(request):
        nonlocal call_count
        call_count += 1
        if call_count < 2:
            return httpx.Response(503)
        return httpx.Response(202, json={
            "status": "ACCEPTED",
            "message_id": "t:0:1",
            "duration_ms": 2,
        })

    with respx.mock:
        respx.post(f"{GATEWAY_URL}/v1/produce").mock(side_effect=side_effect)
        with MzigoProducer(config) as producer:
            result = producer.produce(CONTRACT, PAYLOAD)

    assert result.message_id == "t:0:1"
    assert call_count == 2


@respx.mock
def test_produce_no_retry_on_contract_violation(config):
    call_count = 0

    def side_effect(request):
        nonlocal call_count
        call_count += 1
        return httpx.Response(422, json={
            "status": "REJECTED",
            "topic": "payments.transactions",
            "violations": [{"type": "MISSING_REQUIRED_FIELD", "field": "id", "message": "absent"}],
        })

    with respx.mock:
        respx.post(f"{GATEWAY_URL}/v1/produce").mock(side_effect=side_effect)
        with MzigoProducer(config) as producer:
            with pytest.raises(ContractViolationError):
                producer.produce(CONTRACT, PAYLOAD)

    # Contract violations must not be retried: they will always fail.
    assert call_count == 1


# Async producer

@pytest.mark.asyncio
@respx.mock
async def test_async_produce_accepted(config):
    respx.post(f"{GATEWAY_URL}/v1/produce").mock(
        return_value=httpx.Response(202, json={
            "status": "ACCEPTED",
            "message_id": "payments.transactions:1:99",
            "duration_ms": 4,
        })
    )

    async with AsyncMzigoProducer(config) as producer:
        result = await producer.produce(CONTRACT, PAYLOAD)

    assert result.message_id == "payments.transactions:1:99"


@pytest.mark.asyncio
@respx.mock
async def test_async_produce_violation_raises(config):
    respx.post(f"{GATEWAY_URL}/v1/produce").mock(
        return_value=httpx.Response(422, json={
            "status": "REJECTED",
            "topic": "payments.transactions",
            "violations": [
                {"type": "TYPE_MISMATCH", "field": "amount_cents", "message": "expected number"}
            ],
        })
    )

    async with AsyncMzigoProducer(config) as producer:
        with pytest.raises(ContractViolationError) as exc_info:
            await producer.produce(CONTRACT, PAYLOAD)

    assert exc_info.value.violations[0].type == "TYPE_MISMATCH"


# Context manager cleanup

def test_producer_context_manager_closes_cleanly(config):
    with MzigoProducer(config) as producer:
        assert producer is not None
    # No exception on close
