# Mzigo

**Streaming Data Contracts & Lineage Control Plane for the Lakehouse Era**

Mzigo enforces data contracts at stream ingestion time, tracks column-level lineage across your lakehouse, and gives platform teams blast-radius visibility before a schema change reaches production.

If a producer drops a required field, changes a type incompatibly, or introduces a PII column without a masking policy, Mzigo blocks it at the gateway, not at 3am when your Flink job crashes.

---

## Why This Exists

Schema registries tell you what a schema looks like. They do not tell you:

- Whether that schema change breaks 14 downstream Flink jobs
- Which Iceberg tables will receive malformed data
- Whether a new field contains PII that needs masking before it lands
- Who owns the contract and approved the change
- What the blast radius is if you let it through

Mzigo fills that gap. It sits between your producers and your streaming infrastructure, enforces contracts at runtime, and emits lineage events that power a real governance layer.

---

## Architecture

```
                         ┌─────────────────────────────────────┐
                         │           RUNTIME PLANE              │
                         │                                     │
  Producer SDK ─────────►│  Gateway (Go)                       │
                         │    │                                 │
                         │    ├─ Contract Lookup (hot cache)    │
                         │    ├─ Schema Validation              │
                         │    ├─ PII Field Detection            │
                         │    └─ Masking Policy Apply           │
                         │         │                           │
                         └─────────┼───────────────────────────┘
                                   │
                                   ▼
                              Kafka + Schema Registry
                                   │
                         ┌─────────┼───────────────────────────┐
                         │         │    METADATA PLANE          │
                         │         ▼                           │
                         │  Lineage Worker (Python)            │
                         │    │                                 │
                         │    ├─ OpenLineage event emission     │
                         │    ├─ Marquez sync                   │
                         │    └─ Column-level lineage           │
                         │         │                           │
                         │         ▼                           │
                         │  Blast-Radius Analyzer (Python)     │
                         │    │                                 │
                         │    ├─ DAG traversal                  │
                         │    ├─ Impact scoring                 │
                         │    └─ Breaking-change classification │
                         │         │                           │
                         └─────────┼───────────────────────────┘
                                   │
                         ┌─────────┼───────────────────────────┐
                         │         │    GOVERNANCE PLANE        │
                         │         ▼                           │
                         │  Contract Registry (Go)             │
                         │    │                                 │
                         │    ├─ Contract versioning            │
                         │    ├─ CI/CD gate API                 │
                         │    └─ Policy enforcement (OPA/Rego)  │
                         │         │                           │
                         │         ▼                           │
                         │  Catalog UI (Next.js)               │
                         │    ├─ Contract explorer             │
                         │    ├─ Lineage graph                 │
                         │    └─ Blast-radius dashboard        │
                         └─────────────────────────────────────┘
```

### Runtime Path (latency-critical)

Every message produced through the Mzigo SDK hits the Gateway. The Gateway validates the payload against the active contract, applies any masking policies, and forwards to Kafka. The contract is cached in-process. Validation does not make a network call on the hot path.

Target: **<5ms p99 added latency** on the validation path.

### Metadata Path (eventually consistent)

A separate Lineage Worker consumes from Kafka topics, extracts schema and field-level metadata, and emits OpenLineage events to Marquez. This path is decoupled from the runtime path entirely. A lineage worker failure does not affect message production.

### Governance Path (offline / PR-gated)

Contract changes flow through a CI/CD gate. The `mzigo-contracts` GitHub Action calls the Contracts service to run breaking-change detection and blast-radius analysis before any schema change merges. Changes that would break downstream consumers require explicit approval from affected team owners.

---

## Services

| Service | Language | Role |
|---|---|---|
| `gateway` | Go | Runtime validation, masking, Kafka proxy |
| `contracts` | Go | Contract registry, versioning, CI gate API |
| `lineage` | Python | OpenLineage emission, Marquez integration |
| `analyzer` | Python | Blast-radius, breaking-change detection |
| `masking` | Go | PII detection, dynamic field masking |
| `catalog` | TypeScript/Next.js | Developer UI, contract explorer, lineage graph |

---

## Contract Lifecycle

```
1. Engineer authors a contract YAML in their service repo
2. PR opens → mzigo-contracts GitHub Action runs
3. Action calls Contracts service: breaking-change analysis
4. Blast-radius report posted as PR comment
5. If breaking: affected team owners must approve
6. On merge: contract version registered, gateway cache invalidated
7. At runtime: Gateway enforces the active contract version
8. On violation: message rejected, structured error returned to producer
9. Lineage worker picks up new schema version, updates Marquez
```

---

## Data Flow

```
Producer
  │
  │  (1) Produce message with contract ID + version
  ▼
Gateway
  │  (2) Load contract from local cache
  │  (3) Validate payload schema
  │  (4) Detect PII fields
  │  (5) Apply masking policy
  │  (6) Emit validation span (OpenTelemetry)
  ▼
Kafka Topic
  │
  ├──► Flink Jobs (downstream consumers)
  │
  └──► Lineage Worker
         │  (7) Emit RunEvent to OpenLineage API
         │  (8) Post dataset/job/run facets to Marquez
         ▼
       Marquez
         │
         └──► Catalog UI lineage graph
```

---

## CI/CD Contract Gate

```yaml
# .github/workflows/contract-gate.yml (in producer repo)
- uses: mzigo-io/mzigo-contracts@v1
  with:
    contract: contracts/payments.yaml
    environment: staging
```

The action:
1. Submits the proposed contract diff to the Contracts service
2. Receives a breaking-change classification (SAFE / COMPATIBLE / BREAKING)
3. Fetches blast-radius report: affected topics, jobs, and tables
4. Posts a structured comment on the PR
5. Fails the check if BREAKING and no owner approvals present

---

## Governance Model

Contracts are versioned using a three-field scheme: `MAJOR.MINOR.PATCH`

- **PATCH**: field additions with defaults, description changes, metadata updates
- **MINOR**: new required fields with migration path, new topics
- **MAJOR**: field removals, type changes, rename without alias, semantic reordering

MAJOR bumps require blast-radius analysis and downstream owner sign-off before the gateway will accept the new version in production.

---

## Observability Stack

| Layer | Tool |
|---|---|
| Traces | OpenTelemetry → Tempo |
| Metrics | Prometheus + Grafana |
| Logs | Structured JSON → Loki |
| Lineage | OpenLineage → Marquez |
| Alerts | Alertmanager → PagerDuty |

Every service emits:
- Request duration histograms
- Contract validation counters (pass/fail/version)
- PII detection counters by field classification
- Lineage emission lag

---

## Local Development

Requires: Docker Desktop, `make`

```bash
git clone https://github.com/sogodongo/mzigo
cd mzigo
make dev-up          # starts full local stack
make dev-seed        # loads example contracts and schemas
make dev-produce     # runs example producer against gateway
make dev-lineage     # opens Marquez UI
```

The local stack runs: Kafka, Schema Registry, Postgres, Marquez, Prometheus, Grafana, and all Mzigo services.

---

## Kubernetes Deployment

Each service ships a Helm chart under `infra/helm/`. A single umbrella chart deploys the full platform:

```bash
helm repo add mzigo https://charts.mzigo.io
helm install mzigo mzigo/mzigo-platform \
  --namespace mzigo \
  --values values.production.yaml
```

See [infra/helm/README.md](infra/helm/README.md) for full configuration reference.

---

## Security Model

- All inter-service communication uses mTLS (cert-manager in Kubernetes)
- Gateway authenticates producers via signed JWT with contract scope claims
- PII field classifications are enforced by policy, not by producer declaration
- Contract mutations require audit-logged approval chain
- Rego policies in `policies/` are evaluated by OPA sidecar on the Contracts service
- No secrets in repository; all secrets via Kubernetes Secrets or Vault

---

## Scaling Discussion

**Gateway** is stateless and horizontally scalable. Contract cache is warmed at startup and invalidated via a lightweight pub/sub channel. Under high throughput, validation CPU is the constraint, not I/O.

**Lineage Worker** scales by Kafka partition. One worker per partition group is the operational unit. Lag on the lineage topic does not affect the runtime path.

**Contracts service** is read-heavy in production (CI/CD gate calls, cache fills). Write path (contract registration) is low-frequency. Postgres handles this without read replicas at most scales.

**Blast-radius Analyzer** is the most computationally variable service. Large lineage DAGs with many downstream consumers can take seconds. This is expected and acceptable; it runs offline in CI, not in the request path.

---

## Repository Structure

```
mzigo/
├── services/
│   ├── gateway/         # Go: runtime hot path
│   ├── contracts/       # Go: governance control plane
│   ├── lineage/         # Python: lineage emission
│   ├── analyzer/        # Python: blast-radius analysis
│   ├── masking/         # Go: PII and masking
│   └── catalog/         # TypeScript/Next.js: developer UI
├── sdk/
│   ├── python/          # Python producer SDK
│   └── go/              # Go producer SDK
├── contracts/           # Example and template contracts
├── schemas/             # Canonical schema definitions
├── policies/            # OPA Rego policies
├── infra/
│   ├── terraform/       # Cloud infrastructure
│   ├── helm/            # Kubernetes charts
│   └── docker/          # Local dev compose
├── docs/
│   ├── architecture/    # ADRs and design docs
│   ├── operations/      # Runbooks
│   └── governance/      # Contract model docs
└── .github/workflows/   # CI/CD pipelines
```

---

## Status

| Component | Status |
|---|---|
| Gateway (runtime validation) | In development |
| Contract Registry | In development |
| Lineage Worker | In development |
| Blast-Radius Analyzer | In development |
| PII Masking | In development |
| Catalog UI | Planned |
| Python SDK | In development |
| Helm Charts | Planned |
| Terraform | Planned |

---

## License

Apache 2.0
