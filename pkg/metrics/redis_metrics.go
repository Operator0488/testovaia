package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	RedisQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_query_duration_seconds",
			Help:    "Duration of Redis commands",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command"},
	)

	//open, idle, in_use
	RedisConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "redis_connections",
			Help: "Redis connection pool size by state",
		},
		[]string{"state"},
	)
)

func init() {
	Registry.MustRegister(
		RedisQueryDuration,
		RedisConnections,
	)
}
