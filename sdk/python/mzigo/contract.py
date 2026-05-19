"""
Client-side contract resolution and caching.

The SDK needs to know the contract_id for a topic to include in the
gateway request. Rather than requiring producers to hard-code IDs,
the SDK resolves them from the contracts service and caches the result.

Cache design: simple in-process dict with TTL. The contract registry
does not change frequently. A 5-minute TTL is aggressive enough to
pick up contract version changes without hammering the service.

The cache is keyed by topic name. The contract_id is stable across
versions; only the version string changes when a new version is activated.
"""

from __future__ import annotations

import time
from dataclasses import dataclass

import httpx

from mzigo.types import ContractRef


@dataclass
class _CacheEntry:
    ref: ContractRef
    expires_at: float


class ContractResolver:
    """
    Resolves topic names to ContractRef objects via the contracts service.

    In most deployments, producers will pin their ContractRef explicitly
    (they know their contract_id and version from their service config).
    The resolver is for producers who want dynamic resolution, and for
    the SDK's internal validation that the pinned version matches active.
    """

    def __init__(self, contracts_url: str, ttl_seconds: float = 300.0) -> None:
        self._url = contracts_url.rstrip("/")
        self._ttl = ttl_seconds
        self._cache: dict[str, _CacheEntry] = {}
        self._client = httpx.Client(timeout=5.0)

    def resolve(self, topic: str) -> ContractRef:
        """
        Returns the active ContractRef for a topic.
        Raises ContractNotFoundError if no active contract exists.
        """
        entry = self._cache.get(topic)
        if entry and time.monotonic() < entry.expires_at:
            return entry.ref

        return self._fetch_and_cache(topic)

    def _fetch_and_cache(self, topic: str) -> ContractRef:
        from mzigo.exceptions import ContractNotFoundError, GatewayError

        try:
            response = self._client.get(
                f"{self._url}/internal/v1/contracts/by-topic/{topic}"
            )
        except httpx.RequestError as exc:
            raise GatewayError(f"contracts service unreachable: {exc}") from exc

        if response.status_code == 404:
            raise ContractNotFoundError(topic)

        if not response.is_success:
            raise GatewayError(f"contracts service returned {response.status_code}")

        data = response.json()
        ref = ContractRef(
            topic=topic,
            contract_id=data["contract_id"],
            version=data["version"],
        )

        self._cache[topic] = _CacheEntry(
            ref=ref,
            expires_at=time.monotonic() + self._ttl,
        )
        return ref

    def close(self) -> None:
        self._client.close()
