import pytest
import respx
import httpx

from mzigo.contract import ContractResolver
from mzigo.exceptions import ContractNotFoundError, GatewayError


CONTRACTS_URL = "http://contracts.test"


@pytest.fixture
def resolver() -> ContractResolver:
    r = ContractResolver(CONTRACTS_URL, ttl_seconds=60.0)
    yield r
    r.close()


@respx.mock
def test_resolve_returns_contract_ref(resolver):
    respx.get(f"{CONTRACTS_URL}/internal/v1/contracts/by-topic/payments.transactions").mock(
        return_value=httpx.Response(200, json={
            "contract_id": "abc-123",
            "version": "2.1.0",
        })
    )

    ref = resolver.resolve("payments.transactions")

    assert ref.topic == "payments.transactions"
    assert ref.contract_id == "abc-123"
    assert ref.version == "2.1.0"


@respx.mock
def test_resolve_caches_result(resolver):
    route = respx.get(
        f"{CONTRACTS_URL}/internal/v1/contracts/by-topic/payments.transactions"
    ).mock(return_value=httpx.Response(200, json={
        "contract_id": "abc-123",
        "version": "2.1.0",
    }))

    resolver.resolve("payments.transactions")
    resolver.resolve("payments.transactions")

    # Second call must use cache, not make a second HTTP request
    assert route.call_count == 1


@respx.mock
def test_resolve_404_raises_not_found(resolver):
    respx.get(
        f"{CONTRACTS_URL}/internal/v1/contracts/by-topic/unknown.topic"
    ).mock(return_value=httpx.Response(404))

    with pytest.raises(ContractNotFoundError) as exc_info:
        resolver.resolve("unknown.topic")

    assert exc_info.value.topic == "unknown.topic"


@respx.mock
def test_resolve_service_error_raises_gateway_error(resolver):
    respx.get(
        f"{CONTRACTS_URL}/internal/v1/contracts/by-topic/payments.transactions"
    ).mock(return_value=httpx.Response(500))

    with pytest.raises(GatewayError):
        resolver.resolve("payments.transactions")
