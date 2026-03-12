package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	KafkaMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_total",
			Help: "Total number of Kafka messages processed",
		},
		[]string{"topic", "type"}, // type: produce | consume
	)

	KafkaErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_errors_total",
			Help: "Total number of Kafka errors",
		},
		[]string{"topic", "type"},
	)

	KafkaConsumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Kafka consumer lag by topic, group and partition",
		},
		[]string{"topic"},
	)

	KafkaLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_latency_seconds",
			Help:    "Latency of Kafka operations (produce / consume)",
			Buckets: prometheus.DefBuckets, // стандартные: 5ms, 10ms, 25ms, 50ms, 100ms, ...
		},
		[]string{"topic", "type"},
	)
)

func init() {
	Registry.MustRegister(
		KafkaMessagesTotal,
		KafkaErrorsTotal,
		KafkaConsumerLag,
		KafkaLatencySeconds,
	)
}
