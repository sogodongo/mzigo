from unittest.mock import MagicMock, patch

import pytest

from lineage.emitter import LineageEmitter
from lineage.extractor import FieldExtractor, MessageMetadata


@pytest.fixture
def emitter() -> LineageEmitter:
    return LineageEmitter(
        marquez_url="http://marquez:5000",
        namespace="test",
        emit_timeout=1.0,
    )


@pytest.fixture
def metadata() -> MessageMetadata:
    extractor = FieldExtractor()
    return extractor.extract(
        topic="payments.transactions",
        payload=b'{"transaction_id": "abc", "amount": 100}',
    )


def test_emit_complete_calls_marquez(emitter, metadata):
    with patch.object(emitter._client, "post") as mock_post:
        mock_response = MagicMock()
        mock_response.raise_for_status.return_value = None
        mock_post.return_value = mock_response

        emitter.emit_complete(
            metadata=metadata,
            producer_id="payments-service",
            partition=0,
            offset=42,
        )

        mock_post.assert_called_once()
        call_kwargs = mock_post.call_args

        # Verify the event was sent to the correct endpoint
        assert "/api/v1/lineage" in call_kwargs[0][0]

        event = call_kwargs[1]["json"]
        assert event["eventType"] == "COMPLETE"
        assert event["job"]["name"] == "payments-service"
        assert event["outputs"][0]["name"] == "payments.transactions"


def test_emit_complete_includes_schema_facet(emitter, metadata):
    with patch.object(emitter._client, "post") as mock_post:
        mock_response = MagicMock()
        mock_response.raise_for_status.return_value = None
        mock_post.return_value = mock_response

        emitter.emit_complete(metadata=metadata, producer_id="svc", partition=0, offset=0)

        event = mock_post.call_args[1]["json"]
        output = event["outputs"][0]

        assert "schema" in output["facets"]
        field_names = [f["name"] for f in output["facets"]["schema"]["fields"]]
        assert "transaction_id" in field_names
        assert "amount" in field_names


def test_emit_timeout_does_not_raise(emitter, metadata):
    import httpx

    with patch.object(emitter._client, "post", side_effect=httpx.TimeoutException("timeout")):
        # Should log and increment metric but not raise
        emitter.emit_complete(metadata=metadata, producer_id="svc", partition=0, offset=0)


def test_emit_violation_sends_fail_event(emitter):
    with patch.object(emitter._client, "post") as mock_post:
        mock_response = MagicMock()
        mock_response.raise_for_status.return_value = None
        mock_post.return_value = mock_response

        emitter.emit_violation(
            topic="payments.transactions",
            producer_id="bad-producer",
            violation_type="MISSING_REQUIRED_FIELD",
        )

        event = mock_post.call_args[1]["json"]
        assert event["eventType"] == "FAIL"
        assert "MISSING_REQUIRED_FIELD" in event["run"]["facets"]["errorMessage"]["message"]
