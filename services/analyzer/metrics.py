from prometheus_client import Counter, Histogram

analysis_requests = Counter(
    "mzigo_analyzer_requests_total",
    "Blast-radius analysis requests by topic.",
    ["topic"],
)

analysis_duration = Histogram(
    "mzigo_analyzer_duration_seconds",
    "End-to-end blast-radius analysis time including Marquez fetch and graph traversal.",
    ["topic"],
    buckets=[0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0],
)
