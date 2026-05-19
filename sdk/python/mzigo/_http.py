"""
Internal HTTP transport for the Mzigo SDK.

This module is not part of the public API. It handles the wire protocol
between the SDK and the gateway, including retry logic, error parsing,
and response mapping to typed exceptions.

We use httpx rather than requests because:
- httpx supports both sync and async with the same API surface
- httpx has better timeout handling (connect + read timeouts separately)
- We want a single HTTP dependency, not requests + aiohttp

The transport is internal (_http.py) because we may change the underlying
HTTP library without it being a breaking change to SDK consumers.
"""

from __future__ import annotations

import json
from typing import Any

import httpx
from tenacity import (
    retry,
    retry_if_exception_type,
    stop_after_attempt,
    wait_fixed,
    RetryError,
)

from mzigo.exceptions import (
    ContractNotFoundError,
    ContractViolationError,
    GatewayError,
    GatewayTimeoutError,
    Violation,
)
from mzigo.types import ProduceResult, ProducerConfig


class Transport:
    """
    Synchronous HTTP transport. Used by MzigoProducer.
    """

    def __init__(self, config: ProducerConfig) -> None:
        self._config = config
        self._client = httpx.Client(
            base_url=config.gateway_url.rstrip("/"),
            timeout=httpx.Timeout(
                connect=2.0,
                read=config.timeout,
                write=config.timeout,
                pool=2.0,
            ),
        )

    def produce(
        self,
        topic: str,
        contract_id: str,
        contract_version: str,
        key: str | None,
        payload: dict[str, Any],
    ) -> ProduceResult:
        body = {
            "topic": topic,
            "contract_id": contract_id,
            "contract_version": contract_version,
            "producer_id": self._config.producer_id,
            "key": key or "",
            "payload": payload,
        }

        @retry(
            retry=retry_if_exception_type(GatewayError),
            stop=stop_after_attempt(self._config.max_retries),
            wait=wait_fixed(self._config.retry_wait_seconds),
            reraise=True,
        )
        def _attempt() -> ProduceResult:
            return self._do_produce(body)

        try:
            return _attempt()
        except RetryError as exc:
            raise GatewayError(
                f"gateway unreachable after {self._config.max_retries} attempts"
            ) from exc

    def _do_produce(self, body: dict[str, Any]) -> ProduceResult:
        try:
            response = self._client.post("/v1/produce", json=body)
        except httpx.TimeoutException as exc:
            raise GatewayTimeoutError("gateway request timed out") from exc
        except httpx.RequestError as exc:
            raise GatewayError(f"gateway connection failed: {exc}") from exc

        return _parse_response(response)

    def close(self) -> None:
        self._client.close()

    def __enter__(self) -> "Transport":
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()


class AsyncTransport:
    """
    Async HTTP transport. Used by AsyncMzigoProducer.
    """

    def __init__(self, config: ProducerConfig) -> None:
        self._config = config
        self._client = httpx.AsyncClient(
            base_url=config.gateway_url.rstrip("/"),
            timeout=httpx.Timeout(
                connect=2.0,
                read=config.timeout,
                write=config.timeout,
                pool=2.0,
            ),
        )

    async def produce(
        self,
        topic: str,
        contract_id: str,
        contract_version: str,
        key: str | None,
        payload: dict[str, Any],
    ) -> ProduceResult:
        body = {
            "topic": topic,
            "contract_id": contract_id,
            "contract_version": contract_version,
            "producer_id": self._config.producer_id,
            "key": key or "",
            "payload": payload,
        }

        last_error: Exception | None = None
        for attempt in range(self._config.max_retries):
            try:
                response = await self._client.post("/v1/produce", json=body)
                return _parse_response(response)
            except GatewayError as exc:
                last_error = exc
                continue
            except httpx.TimeoutException as exc:
                last_error = GatewayTimeoutError("gateway request timed out")
                continue
            except httpx.RequestError as exc:
                last_error = GatewayError(f"gateway connection failed: {exc}")
                continue

        raise last_error or GatewayError("all retry attempts failed")

    async def aclose(self) -> None:
        await self._client.aclose()

    async def __aenter__(self) -> "AsyncTransport":
        return self

    async def __aexit__(self, *_: Any) -> None:
        await self.aclose()


def _parse_response(response: httpx.Response) -> ProduceResult:
    """
    Maps gateway response codes and bodies to typed results or exceptions.

    We parse the response body for all non-2xx responses to extract
    structured violation details. A raw status code without context
    is not useful to a producer trying to fix their payload.
    """
    if response.status_code == 202:
        data = response.json()
        return ProduceResult(
            message_id=data.get("message_id", ""),
            topic=data.get("topic", ""),
            duration_ms=data.get("duration_ms", 0),
        )

    if response.status_code == 422:
        data = _safe_json(response)
        raw_violations = data.get("violations", [])
        topic = data.get("topic", "unknown")

        violations = [
            Violation(
                type=v.get("type", "UNKNOWN"),
                field=v.get("field"),
                message=v.get("message", ""),
            )
            for v in raw_violations
        ]

        # No contract for this topic is a special case of 422.
        if any(v.type == "NO_CONTRACT_FOR_TOPIC" for v in violations):
            raise ContractNotFoundError(topic)

        raise ContractViolationError(topic, violations)

    if response.status_code in (500, 502, 503, 504):
        raise GatewayError(f"gateway error {response.status_code}")

    raise GatewayError(f"unexpected gateway response: {response.status_code}")


def _safe_json(response: httpx.Response) -> dict:
    try:
        return response.json()
    except Exception:
        return {}
