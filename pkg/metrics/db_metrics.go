package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of database queries",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query"},
	)

	DBConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_connections",
			Help: "Number of open database connections",
		},
		[]string{"state"}, // state: open | idle | in_use
	)
)

func init() {
	Registry.MustRegister(
		DBQueryDuration,
		DBConnections,
	)
}
