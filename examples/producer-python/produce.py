"""
Example: producing payment transaction events with the Mzigo SDK.

This script demonstrates the complete producer workflow:
contract reference, produce call, error handling, and result logging.

Run against a local dev stack:
    cd mzigo && make dev-up && make dev-seed
    cd examples/producer-python && pip install -e ../../sdk/python
    python produce.py
"""

import sys
import time
import uuid

from mzigo import (
    ContractRef,
    ContractViolationError,
    GatewayError,
    MzigoProducer,
    ProducerConfig,
)

GATEWAY_URL = "http://localhost:8080"
PRODUCER_ID = "example-payments-producer"

CONTRACT = ContractRef(
    topic="payments.transactions",
    contract_id="example-contract-id",
    version="2.1.0",
)


def build_transaction(amount_cents: int, currency: str) -> dict:
    return {
        "transaction_id": str(uuid.uuid4()),
        "account_id": f"acct-{uuid.uuid4().hex[:8]}",
        "amount_cents": amount_cents,
        "currency": currency,
        "status": "AUTHORIZED",
        "produced_at": int(time.time() * 1000),
    }


def main() -> None:
    config = ProducerConfig(
        gateway_url=GATEWAY_URL,
        producer_id=PRODUCER_ID,
    )

    with MzigoProducer(config) as producer:
        print(f"producing 5 transactions to {CONTRACT.topic}")

        for i in range(5):
            payload = build_transaction(amount_cents=(i + 1) * 1000, currency="USD")

            try:
                result = producer.produce(
                    contract=CONTRACT,
                    payload=payload,
                    key=payload["transaction_id"],
                )
                print(f"  accepted  {result.message_id}  ({result.duration_ms}ms)")

            except ContractViolationError as e:
                print(f"  rejected  {e.violations[0].field}: {e.violations[0].message}")
                sys.exit(1)

            except GatewayError as e:
                print(f"  gateway error: {e}")
                sys.exit(1)

        print("done")


if __name__ == "__main__":
    main()
