package application

import (
	"context"
	"errors"

	"git.vepay.dev/knoknok/backend-platform/pkg/config"
	"git.vepay.dev/knoknok/backend-platform/pkg/di"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"git.vepay.dev/knoknok/backend-platform/pkg/s3client"
)

var (
	s3Component = NewComponent("s3", initS3Client, runS3Client)
)

// WithS3 add S3 client component
func WithS3() Option {
	return func(app *Application) error {
		app.components.add(component(s3Component))
		return nil
	}
}

func initS3Client(ctx context.Context, app *Application) error {
	logger.Info(ctx, "S3 initialize",
		logger.String("component", "S3"),
	)
	cfg := config.NewConfigWatcher("S3", app.Env, s3client.NewConfig)

	logger.Info(ctx, "S3 config",
		logger.String("endpoint", cfg.Get().Endpoint),
		logger.String("bucket", cfg.Get().Bucket),
		logger.String("region", cfg.Get().Region),
		logger.Bool("use_ssl", cfg.Get().UseSSL),
		logger.Bool("create_bucket", cfg.Get().CreateBucket),
	)

	client, err := s3client.NewClient(cfg)
	if err != nil {
		return err
	}

	app.Closer.Add(func() error {
		var err error
		defer func() {
			if err != nil {
				logger.Error(ctx, "Component S3 closed with error",
					logger.Err(err),
					logger.String("component", "S3"),
				)
				return
			}
			logger.Info(ctx, "Component closed",
				logger.String("component", "S3"),
			)
		}()

		err = client.Close()
		return err
	})

	app.Health.Add("s3", func(ctx context.Context) error {
		var err error
		defer func() {
			if err == nil {
				return
			}
			logger.Error(ctx, "Health check error",
				logger.Err(err),
				logger.String("component", "S3"),
			)
		}()
		err = client.Ping(ctx)
		return err
	})

	app.Env.Subscribe(cfg)
	app.S3 = client

	di.Register(ctx, app.S3)

	return nil
}

func runS3Client(ctx context.Context, app *Application) error {
	if app.S3 == nil {
		return errors.New("S3 client not initialized")
	}

	return app.S3.Ping(ctx)
}
