"""
Lineage edge store.

Maintains a lightweight cache of (source_topic, consumer) relationships
in Postgres. This is not the full lineage graph; that lives in Marquez.
This table exists to answer one specific query fast: "which consumers
will be affected if I change this topic's contract?"

The contracts service reads this table when computing blast radius.
We upsert here whenever we see a new consumer-topic relationship in
the lineage events we process.
"""

from __future__ import annotations

import asyncpg
import structlog

from metrics import edge_upserts

log = structlog.get_logger(__name__)


class EdgeStore:
    def __init__(self, pool: asyncpg.Pool) -> None:
        self._pool = pool

    async def upsert_edge(
        self,
        source_topic: str,
        consumer_name: str,
        consumer_type: str,
        consumer_team: str | None = None,
    ) -> None:
        """
        Record or refresh a lineage edge.

        last_seen_at is updated on every upsert. The contracts service
        can filter out edges that haven't been seen recently, which handles
        the case where a consumer is decommissioned but its edge record
        persists. We don't delete edges automatically; operators review and
        archive them via the catalog UI.
        """
        try:
            await self._pool.execute(
                """
                INSERT INTO lineage_edges
                    (source_topic, consumer_name, consumer_type, consumer_team, last_seen_at)
                VALUES ($1, $2, $3, $4, now())
                ON CONFLICT (source_topic, consumer_name)
                DO UPDATE SET
                    consumer_type = EXCLUDED.consumer_type,
                    consumer_team = EXCLUDED.consumer_team,
                    last_seen_at = now()
                """,
                source_topic,
                consumer_name,
                consumer_type,
                consumer_team,
            )
            edge_upserts.labels(consumer_type=consumer_type).inc()
        except Exception as exc:
            # Edge upsert failures are logged but not propagated.
            # A failed edge write should not interrupt message processing.
            # The lineage event still emits to Marquez; the Postgres cache
            # will be repopulated on the next occurrence of this edge.
            log.error(
                "edge_upsert_failed",
                source_topic=source_topic,
                consumer_name=consumer_name,
                error=str(exc),
            )
