package application

import (
	"context"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"git.vepay.dev/knoknok/backend-platform/pkg/trace"
)

var (
	traceComponent = NewComponent("trace", initTrace, Noop)
)

// WithTrace добавляет OpenTelemetry трассировку
func WithTrace() Option {
	return func(app *Application) error {
		app.components.add(component(traceComponent))
		return nil
	}
}

// initTrace инициализация OpenTelemetry
func initTrace(ctx context.Context, app *Application) error {

	cfg, err := trace.GetTracingConfig(app.Env)
	if err != nil {
		logger.Error(ctx, "Get config failed", logger.Err(err))
		return err
	}

	logger.Info(ctx, "Tracing initialize")

	shutdown, err := trace.InitProvider(
		ctx,
		cfg)
	if err != nil {
		logger.Error(ctx, "Tracing initialization failed", logger.Err(err))
		return err
	}

	app.Closer.Add(func() error {
		return shutdown(ctx)
	})

	logger.Info(ctx, "Tracing initialized successfully")
	return nil
}
