"""
Field-level metadata extraction from Kafka messages.

The extractor's job is to understand what fields are present in a message
and classify them for lineage emission. This is intentionally separate from
the OpenLineage emitter so each piece is independently testable.

We extract from JSON payloads at this stage. Avro schema-backed extraction
(using the schema registry schema to enumerate fields) is a planned extension
that will give us richer type information without parsing every message.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field


@dataclass
class FieldMetadata:
    path: str
    value_type: str  # string | number | boolean | object | array | null
    nullable: bool
    present: bool


@dataclass
class MessageMetadata:
    topic: str
    schema_id: int | None
    fields: list[FieldMetadata] = field(default_factory=list)
    # Top-level keys observed. Used to detect field presence/absence
    # relative to the contract definition.
    top_level_keys: set[str] = field(default_factory=set)


class FieldExtractor:
    """
    Extracts field-level metadata from a raw message payload.

    For the lineage use case we care about:
    - Which fields are present (for column-level lineage facets)
    - What types were observed (for schema drift detection)
    - Whether any fields are null (for nullability tracking)

    We do not attempt to validate the message here. Validation is the
    gateway's job. We extract what is there regardless of correctness.
    """

    def extract(self, topic: str, payload: bytes, schema_id: int | None = None) -> MessageMetadata:
        try:
            doc = json.loads(payload)
        except (json.JSONDecodeError, UnicodeDecodeError):
            # Non-JSON messages (Avro binary without schema registry framing,
            # Protobuf, etc.) produce empty field metadata. The lineage event
            # still emits with dataset-level information; column-level facets
            # are omitted. This is the correct graceful degradation.
            return MessageMetadata(topic=topic, schema_id=schema_id)

        if not isinstance(doc, dict):
            return MessageMetadata(topic=topic, schema_id=schema_id)

        metadata = MessageMetadata(
            topic=topic,
            schema_id=schema_id,
            top_level_keys=set(doc.keys()),
        )

        self._extract_fields(doc, prefix="", fields=metadata.fields)
        return metadata

    def _extract_fields(self, doc: dict, prefix: str, fields: list[FieldMetadata]) -> None:
        for key, value in doc.items():
            path = f"{prefix}.{key}" if prefix else key
            value_type = self._classify_type(value)

            fields.append(FieldMetadata(
                path=path,
                value_type=value_type,
                nullable=value is None,
                present=True,
            ))

            # Recurse into nested objects for column-level lineage.
            # We cap recursion at 3 levels to prevent pathological inputs
            # (deeply nested metadata blobs) from blowing the stack or
            # producing thousands of field entries per message.
            if isinstance(value, dict) and prefix.count(".") < 3:
                self._extract_fields(value, path, fields)

    @staticmethod
    def _classify_type(value: object) -> str:
        if value is None:
            return "null"
        if isinstance(value, bool):
            return "boolean"
        if isinstance(value, int | float):
            return "number"
        if isinstance(value, str):
            return "string"
        if isinstance(value, list):
            return "array"
        if isinstance(value, dict):
            return "object"
        return "unknown"
