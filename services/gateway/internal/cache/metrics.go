package cache

import "github.com/prometheus/client_golang/prometheus"

type cacheMetrics struct {
	hits           prometheus.Counter
	misses         prometheus.Counter
	fetchErrors    prometheus.Counter
	failOpenAllows prometheus.Counter
}

func newCacheMetrics() *cacheMetrics {
	m := &cacheMetrics{
		hits: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "contract_cache",
			Name:      "hits_total",
			Help:      "Contract cache hits. High ratio indicates healthy cache warm state.",
		}),
		misses: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "contract_cache",
			Name:      "misses_total",
			Help:      "Contract cache misses triggering a fetch from the contracts service.",
		}),
		fetchErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "contract_cache",
			Name:      "fetch_errors_total",
			Help:      "Failed fetches from the contracts service. Alert if sustained.",
		}),
		failOpenAllows: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mzigo_gateway",
			Subsystem: "contract_cache",
			Name:      "fail_open_allows_total",
			Help:      "Messages allowed through on stale contract due to fail_open=true. Always investigate.",
		}),
	}

	prometheus.MustRegister(m.hits, m.misses, m.fetchErrors, m.failOpenAllows)
	return m
}
