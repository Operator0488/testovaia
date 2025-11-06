package trace

import (
	"context"
	"fmt"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc/credentials/insecure"
)

func InitProvider(ctx context.Context, cfg TracingConfig) (shutdown func(context.Context) error, err error) {

	logger.Info(ctx, "Init OpenTelemetry provider",
		logger.String("protocol", cfg.Protocol),
		logger.String("endpoint", cfg.Endpoint),
		logger.Bool("insecure", cfg.Insecure),
		logger.Float64("sampleRatio", cfg.SampleRatio),
		logger.String("serviceName", cfg.ServiceName),
		logger.String("serviceEnv", cfg.ServiceEnv),
		logger.String("serviceVer", cfg.ServiceVer),
	)

	var exp sdktrace.SpanExporter

	switch cfg.Protocol {
	case "http":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
			logger.Info(ctx, "Using insecure HTTP connection for OTLP")
		}
		exp, err = otlptracehttp.New(ctx, opts...)

	default: // grpc по умолчанию
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
			logger.Info(ctx, "Using insecure gRPC connection for OTLP")
		}
		exp, err = otlptracegrpc.New(ctx, opts...)
	}

	if err != nil {
		logger.Error(ctx, "Failed to create OTLP exporter", logger.Err(err))
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	res, _ := resource.New(ctx,
		resource.WithFromEnv(), //атрибуты из переменной
		resource.WithHost(),    //атрибут хоста
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVer),
			semconv.DeploymentEnvironment(cfg.ServiceEnv), //
		),
	)

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exp,
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info(ctx, "TracerProvider initialized")

	return tp.Shutdown, nil
}
