package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// traceProduceMiddleware add trace info to message headers
func traceProduceMiddleware() produceMiddleware {
	return func(ctx context.Context, messages []Message, next produceFunc) error {
		carrier := make(propagation.HeaderCarrier)
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		for _, msg := range messages {
			for k, values := range carrier {
				for _, v := range values {
					msg.Headers = append(msg.Headers, kafka.Header{Key: k, Value: []byte(v)})
				}
			}
		}
		return next(ctx, messages)
	}
}

// healthCheckProducerMiddleware update healthcheck state after successful request
func healthCheckProducerMiddleware(health HealthCheker) produceMiddleware {
	return func(ctx context.Context, messages []Message, next produceFunc) error {
		err := next(ctx, messages)
		if err == nil {
			health.Update(nil)
		}
		return err
	}
}

// traceConsumeMiddleware create new context with trace info from message headers
func traceConsumeMiddleware() consumeMiddleware {
	return func(ctx context.Context, msg Message, next ConsumeHandler) error {
		carrier := make(propagation.HeaderCarrier)
		for _, h := range msg.Headers {
			carrier[h.Key] = append(carrier[h.Key], string(h.Value))
		}
		ctxWithTrace := otel.GetTextMapPropagator().Extract(ctx, carrier)

		return next(ctxWithTrace, msg)
	}
}

// healthCheckProducerMiddleware update healthcheck state after successful request
func healthCheckConsumeMiddleware(health HealthCheker) consumeMiddleware {
	return func(ctx context.Context, message Message, next ConsumeHandler) error {
		err := next(ctx, message)
		if err == nil {
			health.Update(nil)
		}
		return err
	}
}
