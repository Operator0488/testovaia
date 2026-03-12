package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	Registry = prometheus.NewRegistry()
)

// Init регистрирует системные метрики
func Init() {
	Registry.MustRegister(
		collectors.NewGoCollector(),                                       // TODO требует CGO, возможно это нам не подоходит
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}), // TODO требует CGO, возможно это нам не подоходит
	)
}
