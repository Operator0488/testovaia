package application

import (
	"context"
	"fmt"

	"git.vepay.dev/knoknok/backend-platform/pkg/di"
	"git.vepay.dev/knoknok/backend-platform/pkg/kafka"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

var (
	kafkaComponent = NewComponent("kafka", initKafkaClient, runKafkaClient)
)

// WithKafka add kafka client component, available
func WithKafka() Option {
	return func(app *Application) error {
		app.components.add(component(kafkaComponent))
		return nil
	}
}

func initKafkaClient(ctx context.Context, app *Application) error {
	cfg := app.config.GetKafkaConfig()

	logger.Info(ctx, "Kafka initialize")
	client, err := kafka.NewKafkaClient(
		cfg.Brokers,
		cfg.DefaultGroup,
		// TODO TLS + SASL
	)
	if err != nil {
		return err
	}

	app.Closer.Add(client.Close)
	app.Health.Add("kafka", client.HealthCheck)
	app.Kafka = client

	di.Register(ctx, app.Kafka)

	return nil
}

func runKafkaClient(ctx context.Context, app *Application) error {
	if app.Kafka == nil {
		return fmt.Errorf("kafkaClient not initialized")
	}

	return app.Kafka.Run(ctx)
}
