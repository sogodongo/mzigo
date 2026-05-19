"""
Mzigo Lineage Worker

Entry point for the metadata plane lineage worker.
Wires together configuration, database connection, Kafka consumer,
and the OpenLineage emitter.

This process is designed to run as a single-threaded consumer.
Horizontal scaling is achieved by deploying multiple replicas; Kafka's
group coordinator distributes partitions across all live instances.
"""

import asyncio
import logging
import threading
from http.server import HTTPServer

import asyncpg
import structlog
from prometheus_client import MetricsHandler

from config import Settings
from lineage.consumer import LineageConsumer
from lineage.edge_store import EdgeStore
from lineage.emitter import LineageEmitter
from lineage.extractor import FieldExtractor


def configure_logging(level: str) -> None:
    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.processors.add_log_level,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(
            getattr(logging, level.upper(), logging.INFO)
        ),
    )


def start_metrics_server(port: int) -> None:
    server = HTTPServer(("", port), MetricsHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()


async def create_db_pool(database_url: str) -> asyncpg.Pool:
    return await asyncpg.create_pool(
        database_url,
        min_size=2,
        max_size=5,
        command_timeout=10,
    )


async def main_async() -> None:
    settings = Settings()
    configure_logging(settings.log_level)
    log = structlog.get_logger(__name__)

    start_metrics_server(settings.metrics_port)
    log.info("metrics_server_started", port=settings.metrics_port)

    pool = await create_db_pool(settings.database_url)
    log.info("database_pool_created")

    emitter = LineageEmitter(
        marquez_url=settings.openlineage_url,
        namespace=settings.openlineage_namespace,
        emit_timeout=settings.openlineage_emit_timeout_seconds,
    )
    extractor = FieldExtractor()
    edge_store = EdgeStore(pool)

    consumer = LineageConsumer(
        settings=settings,
        emitter=emitter,
        extractor=extractor,
        edge_store=edge_store,
    )

    log.info("lineage_worker_starting")
    # Run the synchronous consumer in the current thread.
    # The async event loop above is used only for the database pool.
    consumer.run()

    await pool.close()


def main() -> None:
    asyncio.run(main_async())


if __name__ == "__main__":
    main()
