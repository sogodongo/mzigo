# ADR-001: Separate Runtime and Metadata Planes

**Status**: Accepted  
**Date**: 2025-01  
**Authors**: Platform Engineering

---

## Context

Mzigo must perform two fundamentally different classes of work:

1. **Synchronous validation** on the producer hot path. Every message passes through this. Latency is directly user-visible. A 50ms spike here means a 50ms spike in every producer's publish latency.

2. **Asynchronous metadata processing** for lineage, blast-radius analysis, and catalog updates. This work is important but not latency-sensitive. A few seconds of lag is operationally acceptable.

The naive design is a single service that does both. This is the wrong call.

---

## Decision

We separate the platform into three distinct planes with no synchronous coupling between them:

**Runtime Plane** (Gateway service, Go)
- Validates messages against contracts
- Applies masking policies
- Forwards to Kafka
- Must not make any network calls to metadata systems on the hot path
- Contract data is cached in-process, warmed at startup, invalidated via async channel

**Metadata Plane** (Lineage + Analyzer services, Python)
- Consumes from Kafka
- Emits OpenLineage events
- Updates Marquez
- Runs blast-radius computations
- Completely decoupled from runtime path

**Governance Plane** (Contracts service, Go)
- Manages contract versions
- Serves the CI/CD gate API
- Enforces policy via OPA
- Handles contract approval workflows
- Read-heavy, not latency-sensitive

---

## Consequences

**Positive**

- Gateway latency is bounded and predictable. Lineage worker outages do not cause producer failures.
- Each plane scales independently. A spike in CI/CD activity does not affect runtime throughput.
- Failure domains are isolated. The most user-visible path (message production) has the fewest dependencies.
- Language choice per plane is justified: Go for latency-sensitive services, Python for metadata and analysis work where ecosystem (networkx, pandas, openlineage-python) matters more than raw throughput.

**Negative**

- Operational complexity: three logical planes means more services to deploy, monitor, and operate.
- Eventual consistency in lineage: there is a lag between a message being produced and its lineage appearing in Marquez. This is acceptable but must be documented to avoid support confusion.
- Contract cache invalidation is an eventually-consistent operation. In the window between a contract update and cache invalidation, the gateway may validate against a slightly stale contract. We accept this tradeoff and bound the staleness window to 30 seconds by default.

---

## Alternatives Considered

**Single monolithic service**

Simplest to deploy and operate. Rejected because coupling the metadata path to the runtime path creates an unacceptable failure dependency. A slow lineage write would add latency to every producer.

**Sidecar per producer**

Each producer runs a local validation sidecar. Eliminates the gateway network hop. Rejected because it makes contract enforcement inconsistent: different producer versions may run different sidecar versions, creating a window where policy changes are not uniformly applied.

**Event sourcing through Kafka for everything including runtime validation**

Producers write to Kafka and a separate consumer validates asynchronously, rejecting bad messages to a DLQ. Rejected because it inverts the failure model in an unacceptable way: producers think they succeeded but their data was silently rejected. Synchronous rejection on the hot path is the correct model for contract enforcement.
