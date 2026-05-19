-- Mzigo local dev database initialization.
--
-- Schema design decisions:
-- - Contracts and their versions are separate tables. A contract is the logical
--   entity (owned by a team, tied to a topic). A contract_version is an
--   immutable snapshot of the schema at a point in time. We never mutate
--   contract_versions after they are created.
--
-- - We store the raw schema as JSONB rather than a foreign key into Schema
--   Registry. This decouples our contract model from the registry implementation
--   and lets us store contracts for topics that don't use Schema Registry.
--
-- - The approval_chain table is append-only. We record every approval and
--   rejection event with timestamp and actor. This is the audit log.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ── Contracts ────────────────────────────────────────────────────────────────

CREATE TABLE contracts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            TEXT NOT NULL UNIQUE,
    topic           TEXT NOT NULL,
    owner_team      TEXT NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE contract_versions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_id     UUID NOT NULL REFERENCES contracts(id),
    version         TEXT NOT NULL,                          -- semver: MAJOR.MINOR.PATCH
    schema_format   TEXT NOT NULL,                          -- AVRO | JSON_SCHEMA | PROTOBUF
    schema_body     JSONB NOT NULL,
    compatibility   TEXT NOT NULL DEFAULT 'BACKWARD',       -- mirrors Schema Registry compat modes
    status          TEXT NOT NULL DEFAULT 'DRAFT',          -- DRAFT | PENDING_APPROVAL | ACTIVE | DEPRECATED
    registry_id     INTEGER,                                -- Schema Registry subject version ID, nullable
    authored_by     TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (contract_id, version)
);

CREATE INDEX idx_contract_versions_contract_id ON contract_versions(contract_id);
CREATE INDEX idx_contract_versions_status ON contract_versions(status);

-- ── Approval Chain ───────────────────────────────────────────────────────────

CREATE TABLE approval_events (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_version_id UUID NOT NULL REFERENCES contract_versions(id),
    actor               TEXT NOT NULL,
    action              TEXT NOT NULL,  -- SUBMITTED | APPROVED | REJECTED | SUPERSEDED
    comment             TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_approval_events_version_id ON approval_events(contract_version_id);

-- ── Policy Assignments ───────────────────────────────────────────────────────

-- Maps field paths within a contract to masking or governance policies.
-- Field paths use dot notation: "payment.card_number"
CREATE TABLE field_policies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_id     UUID NOT NULL REFERENCES contracts(id),
    field_path      TEXT NOT NULL,
    policy_type     TEXT NOT NULL,  -- MASK | REDACT | TOKENIZE | AUDIT_LOG
    policy_config   JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (contract_id, field_path)
);

-- ── Lineage Summary ──────────────────────────────────────────────────────────

-- Lightweight local cache of downstream consumer relationships.
-- The full lineage lives in Marquez; this table powers fast blast-radius queries
-- without requiring a Marquez API call on every contract change check.
CREATE TABLE lineage_edges (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_topic    TEXT NOT NULL,
    consumer_name   TEXT NOT NULL,
    consumer_type   TEXT NOT NULL,  -- FLINK_JOB | ICEBERG_TABLE | KAFKA_CONSUMER | UNKNOWN
    consumer_team   TEXT,
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_topic, consumer_name)
);

CREATE INDEX idx_lineage_edges_source_topic ON lineage_edges(source_topic);

-- ── Violation Log ────────────────────────────────────────────────────────────

-- Gateway writes contract violations here via the contracts service API.
-- This table is the source of truth for violation rate dashboards.
CREATE TABLE contract_violations (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_id         UUID REFERENCES contracts(id),
    contract_version_id UUID REFERENCES contract_versions(id),
    producer_id         TEXT NOT NULL,
    topic               TEXT NOT NULL,
    violation_type      TEXT NOT NULL,  -- SCHEMA_MISMATCH | MISSING_FIELD | TYPE_ERROR | POLICY_VIOLATION
    violation_detail    JSONB NOT NULL,
    occurred_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_violations_contract_id ON contract_violations(contract_id);
CREATE INDEX idx_violations_occurred_at ON contract_violations(occurred_at DESC);
