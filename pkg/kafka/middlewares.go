package kafka

import (
	"context"
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	kafkaProducerTracerName = "kafka-producer"
	kafkaConsumerTracerName = "kafka-consumer"
)

// traceProduceMiddleware add trace info to message headers
func traceProduceMiddleware(topic string) produceMiddleware {
	return func(ctx context.Context, messages []Message, next produceFunc) error {

		tracer := otel.Tracer(kafkaProducerTracerName)
		ctx, span := tracer.Start(ctx, kafkaProducerTracerName+" "+topic)
		defer span.End()

		// Inject trace context в headers
		carrier := make(propagation.HeaderCarrier)
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		for i := range messages {
			for k, values := range carrier {
				for _, v := range values {
					messages[i].Headers = append(messages[i].Headers,
						kafka.Header{Key: k, Value: []byte(v)})
				}
			}
		}

		err := next(ctx, messages)
		if err != nil {
			span.RecordError(err)
		}
		return err
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
func traceConsumeMiddleware(topic string) consumeMiddleware {
	return func(ctx context.Context, msg Message, next ConsumeHandler) error {
		carrier := make(propagation.HeaderCarrier)
		for _, h := range msg.Headers {
			carrier[h.Key] = append(carrier[h.Key], string(h.Value))
		}
		ctxWithTrace := otel.GetTextMapPropagator().Extract(ctx, carrier)

		tracer := otel.Tracer(kafkaConsumerTracerName)
		ctx, span := tracer.Start(ctx, kafkaConsumerTracerName+" "+topic)
		defer span.End()

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

func metricsProduceMiddleware() produceMiddleware {
	return func(ctx context.Context, messages []Message, next produceFunc) error {

		var topic string

		if len(messages) > 0 {
			topic = messages[0].Topic
		}

		start := time.Now()
		err := next(ctx, messages)
		duration := time.Since(start)

		metrics.KafkaLatencySeconds.WithLabelValues(topic, "produce").Observe(duration.Seconds())

		if err != nil {
			metrics.KafkaErrorsTotal.WithLabelValues(topic, "produce").Inc()
			return err
		}

		metrics.KafkaMessagesTotal.WithLabelValues(topic, "produce").Add(float64(len(messages)))

		return nil
	}
}

func metricsConsumeMiddleware() consumeMiddleware {
	return func(ctx context.Context, msg Message, next ConsumeHandler) error {

		topic := msg.Topic

		start := time.Now()
		err := next(ctx, msg)
		duration := time.Since(start)

		metrics.KafkaLatencySeconds.WithLabelValues(topic, "consume").Observe(duration.Seconds())

		if err != nil {
			metrics.KafkaErrorsTotal.WithLabelValues(topic, "consume").Inc()
			return err
		}

		metrics.KafkaMessagesTotal.WithLabelValues(topic, "consume").Inc()

		return nil
	}
}
