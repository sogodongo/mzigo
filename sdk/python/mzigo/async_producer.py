"""
AsyncMzigoProducer: asyncio-native variant for async producers.

The async producer is functionally identical to MzigoProducer but uses
httpx.AsyncClient under the hood and exposes coroutine methods.

Usage:

    from mzigo import AsyncMzigoProducer, ContractRef, ProducerConfig

    config = ProducerConfig(
        gateway_url="https://mzigo-gateway.internal",
        producer_id="async-service",
    )
    contract = ContractRef(
        topic="events.clicks",
        contract_id="e5f6g7h8",
        version="1.0.0",
    )

    async with AsyncMzigoProducer(config) as producer:
        result = await producer.produce(contract=contract, payload={...})
"""

from __future__ import annotations

from typing import Any

from mzigo._http import AsyncTransport
from mzigo.types import ContractRef, ProduceResult, ProducerConfig


class AsyncMzigoProducer:
    """
    Async contract-aware Kafka producer.

    The async producer shares no state with MzigoProducer. Both can be
    used in the same process without conflict.
    """

    def __init__(self, config: ProducerConfig) -> None:
        self._config = config
        self._transport = AsyncTransport(config)

    async def produce(
        self,
        contract: ContractRef,
        payload: dict[str, Any],
        key: str | None = None,
    ) -> ProduceResult:
        return await self._transport.produce(
            topic=contract.topic,
            contract_id=contract.contract_id,
            contract_version=contract.version,
            key=key,
            payload=payload,
        )

    async def aclose(self) -> None:
        await self._transport.aclose()

    async def __aenter__(self) -> "AsyncMzigoProducer":
        return self

    async def __aexit__(self, *_: Any) -> None:
        await self.aclose()
