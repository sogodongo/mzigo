package handler

import "github.com/prometheus/client_golang/prometheus"

type maskMetrics struct {
	fieldsTransformed *prometheus.CounterVec
	failures          *prometheus.CounterVec
	patternWarnings   *prometheus.CounterVec
	latency           *prometheus.HistogramVec
}

func newMaskMetrics() *maskMetrics {
	m := &maskMetrics{
		fieldsTransformed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mzigo_masking",
			Name:      "fields_transformed_total",
			Help:      "Total fields masked by topic. Useful for auditing masking volume.",
		}, []string{"topic"}),

		failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mzigo_masking",
			Name:      "failures_total",
			Help:      "Messages that could not be masked. Always investigate: these result in blocked messages.",
		}, []string{"topic"}),

		// patternWarnings indicate fields that look like PII but lack a
		// contract policy. A nonzero rate here means contracts need updating.
		patternWarnings: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mzigo_masking",
			Name:      "pattern_warnings_total",
			Help:      "PII-like fields detected by pattern matching without a contract policy declaration.",
		}, []string{"topic"}),

		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "mzigo_masking",
			Name:      "duration_milliseconds",
			Help:      "Masking operation latency. Should stay well below 2ms on typical payloads.",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 25},
		}, []string{"topic"}),
	}

	prometheus.MustRegister(
		m.fieldsTransformed, m.failures,
		m.patternWarnings, m.latency,
	)
	return m
}
