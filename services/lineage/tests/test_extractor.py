import pytest

from lineage.extractor import FieldExtractor


@pytest.fixture
def extractor() -> FieldExtractor:
    return FieldExtractor()


def test_flat_json_extracts_all_fields(extractor):
    payload = b'{"transaction_id": "abc", "amount": 100, "currency": "USD"}'
    metadata = extractor.extract("payments.transactions", payload)

    paths = {f.path for f in metadata.fields}
    assert "transaction_id" in paths
    assert "amount" in paths
    assert "currency" in paths


def test_nested_fields_are_extracted(extractor):
    payload = b'{"payment": {"card": {"last_four": "4242"}}}'
    metadata = extractor.extract("payments.transactions", payload)

    paths = {f.path for f in metadata.fields}
    assert "payment" in paths
    assert "payment.card" in paths
    assert "payment.card.last_four" in paths


def test_null_field_is_marked_nullable(extractor):
    payload = b'{"account_id": null}'
    metadata = extractor.extract("test.topic", payload)

    field = next(f for f in metadata.fields if f.path == "account_id")
    assert field.nullable is True
    assert field.value_type == "null"


def test_invalid_json_returns_empty_metadata(extractor):
    payload = b"\x00\x01\x02binary avro data"
    metadata = extractor.extract("test.topic", payload)

    assert metadata.fields == []
    assert metadata.topic == "test.topic"


def test_type_classification(extractor):
    payload = b'{"s": "hello", "n": 42, "b": true, "a": [1,2], "o": {}, "z": null}'
    metadata = extractor.extract("test.topic", payload)

    types = {f.path: f.value_type for f in metadata.fields if "." not in f.path}
    assert types["s"] == "string"
    assert types["n"] == "number"
    assert types["b"] == "boolean"
    assert types["a"] == "array"
    assert types["o"] == "object"
    assert types["z"] == "null"


def test_top_level_keys_populated(extractor):
    payload = b'{"id": "x", "amount": 50}'
    metadata = extractor.extract("test.topic", payload)

    assert metadata.top_level_keys == {"id", "amount"}


def test_recursion_depth_capped(extractor):
    # Build a deeply nested object: level1.level2.level3.level4.level5
    payload = b'{"l1": {"l2": {"l3": {"l4": {"l5": "deep"}}}}}'
    metadata = extractor.extract("test.topic", payload)

    paths = {f.path for f in metadata.fields}
    # Recursion is capped at 3 dots deep; l4 and beyond should not appear
    assert "l1.l2.l3.l4" not in paths
