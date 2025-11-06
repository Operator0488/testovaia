package application

import (
	"context"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	metricsComponent = NewComponent("metrics", initMetrics, runMetrics)
)

func WithMetrics() Option {
	return func(app *Application) error {
		app.components.add(component(metricsComponent))
		return nil
	}
}

func initMetrics(ctx context.Context, app *Application) error {
	logger.Info(ctx, "Metrics initializing")
	metrics.Init()
	return nil
}

func runMetrics(ctx context.Context, app *Application) error {
	logger.Info(ctx, "Metrics endpoint starting")

	addr := app.config.GetMetricsAddr()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	app.Closer.Add(func() error {
		logger.Info(ctx, "Metrics server shutting down")
		return server.Shutdown(ctx)
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "Metrics server failed", logger.Err(err))
			app.stop()
		}
	}()

	return nil
}
