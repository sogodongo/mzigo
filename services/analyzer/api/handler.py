"""
Analyzer API handlers.

Two endpoints:

POST /v1/blast-radius
  Called by the contracts service gate handler and the catalog UI.
  Accepts a topic and a list of changed fields, returns a full blast-radius report.
  This is the primary endpoint and the one the CI gate comment is built from.

GET /v1/blast-radius/{topic}
  Returns a cached blast-radius report for a topic using its currently
  active changed fields. Primarily used by the catalog UI for the "what
  would change if I modified this topic?" exploration view.
"""

from __future__ import annotations

import time
from typing import Any

import structlog
from fastapi import FastAPI, HTTPException, Request, Response
from pydantic import BaseModel

from analyzer.graph import LineageGraphBuilder
from analyzer.report import ReportAssembler
from analyzer.scorer import ImpactScorer
from analyzer.traversal import DAGTraverser
from config import Settings
from metrics import analysis_duration, analysis_requests

log = structlog.get_logger(__name__)


class BlastRadiusRequest(BaseModel):
    topic: str
    changed_fields: list[str]
    # added_fields is informational; included in the report but lower severity
    added_fields: list[str] = []


def create_app(settings: Settings) -> FastAPI:
    app = FastAPI(
        title="Mzigo Analyzer",
        description="Blast-radius analysis for streaming data contract changes",
        version="0.1.0",
        docs_url="/docs",
        redoc_url=None,
    )

    graph_builder = LineageGraphBuilder(
        marquez_url=settings.marquez_url,
        namespace=settings.marquez_namespace,
        max_depth=settings.max_traversal_depth,
    )
    traverser = DAGTraverser()
    scorer = ImpactScorer()
    assembler = ReportAssembler()

    # In-memory report cache. Keyed by (topic, frozenset(changed_fields)).
    # Bounded by the number of unique (topic, change_set) combinations in
    # active CI pipelines, which is small in practice.
    _cache: dict[tuple, tuple[float, Any]] = {}

    @app.get("/healthz")
    async def healthz() -> dict:
        return {"status": "ok"}

    @app.get("/readyz")
    async def readyz() -> dict:
        return {"status": "ok"}

    @app.post("/v1/blast-radius")
    async def compute_blast_radius(req: BlastRadiusRequest) -> dict:
        cache_key = (req.topic, frozenset(req.changed_fields))
        cached_at, cached_report = _cache.get(cache_key, (0, None))

        if cached_report and (time.monotonic() - cached_at) < settings.report_cache_ttl_seconds:
            log.debug("blast_radius_cache_hit", topic=req.topic)
            return cached_report

        analysis_requests.labels(topic=req.topic).inc()
        start = time.monotonic()

        try:
            report = _run_analysis(
                graph_builder, traverser, scorer, assembler,
                settings, req,
            )
        except Exception as exc:
            log.error("blast_radius_analysis_failed", topic=req.topic, error=str(exc))
            raise HTTPException(status_code=500, detail="Analysis failed") from exc
        finally:
            analysis_duration.labels(topic=req.topic).observe(time.monotonic() - start)

        result = report.to_dict()
        _cache[cache_key] = (time.monotonic(), result)

        log.info(
            "blast_radius_computed",
            topic=req.topic,
            changed_fields=req.changed_fields,
            affected_consumers=report.total_consumers_affected,
            worst_impact=report.worst_impact.value,
        )

        return result

    return app


def _run_analysis(
    graph_builder: LineageGraphBuilder,
    traverser: DAGTraverser,
    scorer: ImpactScorer,
    assembler: ReportAssembler,
    settings: Settings,
    req: BlastRadiusRequest,
) -> Any:
    graph = graph_builder.build_for_topic(req.topic)

    if graph.number_of_nodes() == 0:
        log.info("no_lineage_found", topic=req.topic)
        from analyzer.report import BlastRadiusReport
        from analyzer.scorer import ImpactLevel
        from datetime import datetime, timezone
        return BlastRadiusReport(
            topic=req.topic,
            changed_fields=req.changed_fields,
            generated_at=datetime.now(tz=timezone.utc).isoformat(),
            total_consumers_affected=0,
            worst_impact=ImpactLevel.NONE,
            consumers=[],
            summary=f"No lineage found for topic {req.topic}. "
                    f"This may indicate the topic has no recorded consumers yet.",
        )

    source_id = traverser.find_source_node_id(
        graph, req.topic, settings.marquez_namespace
    )

    if not source_id:
        log.warning("source_node_not_found", topic=req.topic)
        raise ValueError(f"Topic {req.topic!r} not found in lineage graph")

    reachable = traverser.find_reachable(
        graph, source_id, settings.max_traversal_depth
    )

    scored = scorer.score(reachable, set(req.changed_fields))

    return assembler.assemble(req.topic, req.changed_fields, scored)
