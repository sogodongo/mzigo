package handler

import "github.com/prometheus/client_golang/prometheus"

type produceMetrics struct {
	accepted             *prometheus.CounterVec
	rejected             *prometheus.CounterVec
	produceErrors        *prometheus.CounterVec
	versionMismatches    *prometheus.CounterVec
	validationErrors     prometheus.Counter
	contractLookupErrors prometheus.Counter
	latency              *prometheus.HistogramVec
}

func newProduceMetrics() *produceMetrics {
	m := &produceMetrics{
		accepted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "produce",
			Name:      "accepted_total",
			Help:      "Messages accepted and forwarded to Kafka, by topic.",
		}, []string{"topic"}),

		rejected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "produce",
			Name:      "rejected_total",
			Help:      "Messages rejected by contract validation, by topic and violation type.",
		}, []string{"topic", "violation_type"}),

		produceErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "produce",
			Name:      "kafka_errors_total",
			Help:      "Kafka produce failures after validation passed. Alert if nonzero.",
		}, []string{"topic"}),

		versionMismatches: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "produce",
			Name:      "version_mismatches_total",
			Help:      "Producer declared a contract version that differs from the active version.",
		}, []string{"topic"}),

		validationErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "produce",
			Name:      "validation_processing_errors_total",
			Help:      "Internal errors during validation processing. Distinct from contract violations.",
		}),

		contractLookupErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "produce",
			Name:      "contract_lookup_errors_total",
			Help:      "Failed contract lookups. If sustained, check the contracts service and cache.",
		}),

		// Buckets are sized for the <5ms p99 target. The 10ms and 25ms buckets
		// are canaries: sustained counts there indicate a performance regression.
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "produce",
			Name:      "duration_milliseconds",
			Help:      "End-to-end produce handler latency including validation and Kafka produce.",
			Buckets:   []float64{1, 2, 5, 10, 25, 50, 100, 250},
		}, []string{"topic"}),
	}

	prometheus.MustRegister(
		m.accepted, m.rejected, m.produceErrors,
		m.versionMismatches, m.validationErrors,
		m.contractLookupErrors, m.latency,
	)
	return m
}
