from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="MZIGO_ANALYZER_",
        env_file=".env",
    )

    # Marquez
    marquez_url: str = Field(..., description="Marquez API base URL")
    marquez_namespace: str = Field(default="mzigo")
    # How many hops to traverse when building the blast-radius DAG.
    # 5 covers the realistic depth of most production lineage graphs.
    # Deeper traversal is possible but rarely changes the impact assessment.
    max_traversal_depth: int = Field(default=5)

    # Postgres (for the lineage edge cache, used when Marquez is unavailable)
    database_url: str = Field(..., description="PostgreSQL connection string")

    # API server
    host: str = Field(default="0.0.0.0")
    port: int = Field(default=8083)
    metrics_port: int = Field(default=9103)

    # How long to cache a computed blast-radius report.
    # Reports are expensive (multi-hop Marquez queries + graph traversal).
    # 60s cache means CI pipelines that trigger multiple checks in quick
    # succession reuse the same report rather than re-traversing the DAG.
    report_cache_ttl_seconds: int = Field(default=60)

    log_level: str = Field(default="info")
