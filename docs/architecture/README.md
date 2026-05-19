# Architecture Documentation

## Architecture Decision Records

| ADR | Title | Status |
|---|---|---|
| [ADR-001](adr/ADR-001-plane-separation.md) | Separate Runtime and Metadata Planes | Accepted |
| [ADR-002](adr/ADR-002-language-selection.md) | Language Selection Per Service | Accepted |

## Design Documents

- [Runtime Flow](runtime-flow.md) - Request lifecycle from producer to Kafka
- [Contract Lifecycle](contract-lifecycle.md) - From authoring to enforcement
- [Lineage Model](lineage-model.md) - How column-level lineage is tracked
- [Blast Radius Analysis](blast-radius.md) - DAG traversal and impact scoring

## Diagrams

Architecture diagrams are maintained as code in this directory using Mermaid.
They render inline in GitHub and in the catalog UI.
