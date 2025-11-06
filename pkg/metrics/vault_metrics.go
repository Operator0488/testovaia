package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	VaultSecretAccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vault_secret_access_total",
			Help: "Total number of Vault secret access attempts",
		},
		[]string{"type", "mount", "path"},
	)

	VaultErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vault_errors_total",
			Help: "Total number of Vault access errors",
		},
		[]string{"type", "mount", "path"},
	)

	VaultRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vault_request_duration_seconds",
			Help:    "Duration of Vault requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type", "mount", "path"},
	)
)

func init() {
	Registry.MustRegister(
		VaultSecretAccessTotal,
		VaultErrorsTotal,
		VaultRequestDurationSeconds,
	)
}
