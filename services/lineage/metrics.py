from prometheus_client import Counter, Gauge, Histogram

# Message processing counters
messages_processed = Counter(
    "mzigo_lineage_messages_processed_total",
    "Messages consumed and processed from Kafka, by topic.",
    ["topic"],
)

messages_failed = Counter(
    "mzigo_lineage_messages_failed_total",
    "Messages that failed processing after all retries, by failure reason.",
    ["topic", "reason"],
)

# OpenLineage emission
lineage_events_emitted = Counter(
    "mzigo_lineage_events_emitted_total",
    "OpenLineage RunEvents successfully emitted to Marquez.",
    ["event_type"],  # START | COMPLETE | FAIL
)

lineage_emit_failures = Counter(
    "mzigo_lineage_emit_failures_total",
    "Failed OpenLineage event emissions. Marquez unavailability lands here.",
)

# Processing latency
processing_duration = Histogram(
    "mzigo_lineage_processing_duration_seconds",
    "Time to process a single message from consume to lineage emit.",
    buckets=[0.01, 0.05, 0.1, 0.5, 1.0, 5.0],
)

# Consumer lag: difference between latest Kafka offset and committed offset.
# Sustained lag indicates the worker is falling behind production volume.
consumer_lag = Gauge(
    "mzigo_lineage_consumer_lag",
    "Estimated consumer lag in messages, by topic and partition.",
    ["topic", "partition"],
)

# Edge cache upserts
edge_upserts = Counter(
    "mzigo_lineage_edge_upserts_total",
    "Lineage edge records upserted into Postgres, by consumer type.",
    ["consumer_type"],
)
