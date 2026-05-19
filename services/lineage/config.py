from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="MZIGO_LINEAGE_",
        env_file=".env",
        env_file_encoding="utf-8",
    )

    # Kafka
    kafka_bootstrap_servers: str = Field(..., description="Kafka broker addresses")
    kafka_consumer_group: str = Field(default="mzigo-lineage-worker")
    kafka_topics: list[str] = Field(
        default_factory=lambda: ["mzigo.lineage.events"],
        description="Topics to consume for lineage extraction",
    )
    # How many messages to process before committing offsets.
    # Higher values improve throughput but increase reprocessing on restart.
    kafka_commit_interval: int = Field(default=100)
    kafka_poll_timeout_seconds: float = Field(default=1.0)

    # OpenLineage / Marquez
    openlineage_url: str = Field(..., description="Marquez API base URL")
    openlineage_namespace: str = Field(default="mzigo")
    # Emit timeout bounds how long we wait for Marquez to accept an event.
    # Marquez outages should not block the consumer loop.
    openlineage_emit_timeout_seconds: float = Field(default=5.0)

    # Postgres (for lineage edge cache)
    database_url: str = Field(..., description="PostgreSQL connection string")

    # Metrics
    metrics_port: int = Field(default=9102)

    # Logging
    log_level: str = Field(default="info")

    @field_validator("kafka_topics", mode="before")
    @classmethod
    def parse_topics(cls, v: str | list) -> list[str]:
        # Allow comma-separated string from environment variable
        if isinstance(v, str):
            return [t.strip() for t in v.split(",")]
        return v
