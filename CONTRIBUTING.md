# Contributing to Mzigo

This document covers how to get the platform running locally, how the codebase is organized, and what the expectations are for contributions.

## Getting Started

Requirements: Docker Desktop, Go 1.22, Python 3.11, Node 20, Make.

```bash
git clone https://github.com/sogodongo/mzigo
cd mzigo
make dev-up     # Start the full local infrastructure stack
make dev-seed   # Load example contracts
```

The local stack starts Kafka, Schema Registry, Postgres, Marquez, Prometheus, Grafana, Loki, and the OpenTelemetry Collector. First startup takes a few minutes while images pull.

Verify everything is running:

```bash
make dev-status
```

## Repository Layout

Each service is independently buildable and testable. There are no cross-service Go module imports. Python services have their own virtual environments.

```
services/gateway/     # Go: runtime validation hot path
services/contracts/   # Go: governance control plane
services/masking/     # Go: PII detection and masking
services/lineage/     # Python: OpenLineage emission
services/analyzer/    # Python: blast-radius analysis
services/catalog/     # TypeScript/Next.js: developer UI
sdk/python/           # Python: producer SDK
infra/terraform/      # AWS infrastructure modules
infra/helm/           # Kubernetes Helm charts
infra/docker/         # Local dev compose stack
docs/architecture/    # ADRs and design documents
```

## Running Tests

```bash
make test          # All services
make test-go       # Go services only
make test-python   # Python services only
```

Individual service tests:

```bash
cd services/gateway && go test ./... -race
cd services/analyzer && pytest tests/ -v
```

## Code Standards

**Go services**

- `golangci-lint` must pass. Run `make lint-go` before pushing.
- Structured logging via `zerolog`. No `fmt.Println` in service code.
- All exported functions have doc comments.
- Errors are wrapped with context: `fmt.Errorf("doing X: %w", err)`.
- No `utils` or `helpers` packages. If you need one, the abstraction is wrong.

**Python services**

- `ruff` must pass. Run `make lint-python` before pushing.
- Type annotations on all function signatures.
- `structlog` for structured logging. No bare `print()`.
- Pydantic for configuration and data validation.

**TypeScript**

- `tsc --noEmit` must pass. Run `make type-check` in `services/catalog/`.
- No `any` types unless unavoidable, with a comment explaining why.

**All files**

- No em dashes anywhere. Use plain hyphens or restructure the sentence.
- Comments explain why, not what. Delete comments that describe obvious code.
- No `TODO` comments in merged code. Open an issue instead.

## Adding a New Service

1. Create the directory under `services/`.
2. Add a `Dockerfile` that produces a non-root image with a health check endpoint.
3. Add a `Chart.yaml` and `values.yaml` under `infra/helm/charts/`.
4. Add the service to the CI workflow in `.github/workflows/ci.yml`.
5. Add CODEOWNERS entry in `.github/CODEOWNERS`.
6. Add a Prometheus `ServiceMonitor` in the Helm chart templates.
7. Document the service's responsibilities in its own `README.md`.

## Changing a Data Contract

Contract changes affect downstream consumers. The process exists to prevent silent breakage:

1. Edit the contract YAML in `contracts/`.
2. Open a PR. The `contract-gate` CI action runs automatically.
3. Read the impact report posted as a PR comment.
4. If the change is BREAKING, wait for approval from affected team owners before merging.
5. On merge, the gateway cache is invalidated and updated automatically.

Do not bypass the gate. If it is blocking a legitimate change, fix the contract or get the required approvals.

## Writing Architecture Decision Records

When a decision has lasting architectural impact, document it as an ADR:

1. Copy `docs/architecture/adr/ADR-000-template.md` to a new file.
2. Number sequentially: `ADR-003-your-decision.md`.
3. Fill in: context, decision, consequences, alternatives considered.
4. Add it to the ADR index in `docs/architecture/README.md`.

ADRs are immutable once merged. If a decision changes, write a new ADR that supersedes the old one.

## Commit Messages

Format: `type(scope): summary`

Types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`

Scope: the service or area affected: `gateway`, `contracts`, `lineage`, `infra`, `sdk`, `docs`

The summary line explains what changed and why it matters to the platform, not what files were touched. Keep it under 72 characters.

Good:
```
feat(gateway): add fail-open per-contract cache policy
fix(lineage): prevent duplicate emission during consumer rebalance
chore(infra): pin MSK broker instance type in staging tfvars
```

Not good:
```
fix bug
update gateway
WIP
```

## Opening Issues

Use the issue templates for bugs and feature requests. Include enough context that someone unfamiliar with your setup can reproduce the problem or understand the request.

For security issues, do not open a public issue. Email the maintainers directly.

## Local Ports

| Service | Port |
|---|---|
| Gateway API | 8080 |
| Contracts API | 8081 |
| Analyzer API | 8083 |
| Masking API | 8084 |
| Catalog UI | 3000 |
| Marquez UI | 3001 |
| Grafana | 3002 |
| Prometheus | 9090 |
| Kafka | 9092 |
| Schema Registry | 8085 |
| Postgres | 5432 |
