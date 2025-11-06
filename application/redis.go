package application

import (
	"context"

	"git.vepay.dev/knoknok/backend-platform/pkg/di"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"git.vepay.dev/knoknok/backend-platform/pkg/redis"
)

var (
	redisComponent = NewComponent("redis", initRedisClient, Noop)
)

func WithRedis() Option {
	return func(app *Application) error {
		app.components.add(component(redisComponent))
		return nil
	}
}

func initRedisClient(ctx context.Context, app *Application) error {
	cfg := app.config.GetRedisConfig()

	logger.Info(ctx, "Redis initialize")
	client, err := redis.New(ctx, cfg)
	if err != nil {
		return err
	}

	app.Closer.Add(client.Close)
	app.Health.Add("redis", client.HealthCheck)
	app.Redis = client

	di.Register(ctx, app.Redis)
	return nil
}
