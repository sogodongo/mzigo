# ADR-002: Language Selection Per Service

**Status**: Accepted  
**Date**: 2025-01  
**Authors**: Platform Engineering

---

## Context

Mzigo spans multiple distinct engineering domains: high-throughput message validation, graph-based lineage analysis, metadata processing, and a developer UI. Each domain has different performance characteristics and ecosystem requirements. Using a single language across all services would force tradeoffs that hurt the critical paths.

---

## Decision

**Go** for Gateway, Contracts, and Masking services.

These services are either on the runtime hot path or serve high-frequency synchronous API traffic. Go's predictable latency profile, low memory overhead, and strong concurrency primitives are the right fit. The compiled binary with no runtime dependency also simplifies container image management and startup time in Kubernetes.

**Python** for Lineage Worker and Blast-Radius Analyzer.

These services are not latency-sensitive. Their constraints are ecosystem access and analytical expressiveness. `openlineage-python` is the reference client for OpenLineage event emission. `networkx` provides the graph traversal primitives needed for blast-radius DAG analysis. Rewriting these in Go would mean maintaining custom implementations of mature Python libraries with no runtime benefit.

**TypeScript / Next.js** for Catalog UI.

The catalog is a developer-facing web interface. TypeScript gives type safety across the full stack. Next.js handles routing, SSR for fast initial loads, and API routes for BFF (backend-for-frontend) calls. The React ecosystem provides the graph visualization components (React Flow) needed for the lineage UI.

---

## Consequences

**Positive**

- Each service uses the language most suited to its constraints.
- No performance compromises on the hot path to satisfy ecosystem needs on the metadata path.
- Python services can consume the full OpenLineage and analytics ecosystem without adaptation layers.

**Negative**

- Polyglot monorepo requires engineers to context-switch between Go and Python.
- Shared data structures (contract schema, error types) must be defined separately per language and kept in sync. We mitigate this with Protobuf definitions in `schemas/protobuf/` as the canonical source.
- CI pipeline must support Go, Python, and Node toolchains.

---

## Alternatives Considered

**All Go**

Cleanest operationally. Rejected because re-implementing OpenLineage client, networkx-equivalent graph analysis, and a viable frontend in Go is significant non-differentiating engineering work.

**All Python (FastAPI)**

Ecosystem fit for metadata services. Rejected because Python's GIL and latency profile make it unsuitable for the Gateway's validation hot path under high message throughput.

**Rust for Gateway**

Better raw performance than Go. Rejected for now: the engineering cost of Rust in a polyglot team, combined with the fact that Go's latency profile already meets our <5ms p99 target, does not justify the tradeoff. Rust remains a valid future migration path if Go becomes a bottleneck at extreme scale.
