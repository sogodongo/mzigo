package handler

import "github.com/prometheus/client_golang/prometheus"

type gateMetrics struct {
	classifications *prometheus.CounterVec
}

func newGateMetrics() *gateMetrics {
	m := &gateMetrics{
		classifications: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mzigo_contracts",
			Subsystem: "gate",
			Name:      "classifications_total",
			Help:      "CI gate check outcomes by classification. Rising BREAKING count warrants team communication.",
		}, []string{"classification"}),
	}
	prometheus.MustRegister(m.classifications)
	return m
}
