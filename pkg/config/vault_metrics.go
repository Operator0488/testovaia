package config

import (
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	"strings"
	"time"
)

func withMetrics[T any](typ, mount, path string, f func() (T, error)) (T, error) {
	start := time.Now()
	val, err := f()
	metrics.VaultRequestDurationSeconds.WithLabelValues(typ, mount, path).
		Observe(time.Since(start).Seconds())
	metrics.VaultSecretAccessTotal.WithLabelValues(typ, mount, path).Inc()
	if err != nil {
		metrics.VaultErrorsTotal.WithLabelValues(typ, mount, path).Inc()
	}
	return val, err
}

func withMetrics1(typ, mount, path string, f func() error) error {
	start := time.Now()
	err := f()
	metrics.VaultRequestDurationSeconds.WithLabelValues(typ, mount, path).
		Observe(time.Since(start).Seconds())
	metrics.VaultSecretAccessTotal.WithLabelValues(typ, mount, path).Inc()
	if err != nil {
		metrics.VaultErrorsTotal.WithLabelValues(typ, mount, path).Inc()
	}
	return err
}

func mountFromPath(path string) string {
	p := strings.TrimPrefix(path, "/")
	if p == "" {
		return "-"
	}
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}
