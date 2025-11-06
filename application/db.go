package application

import (
	"context"
	"fmt"

	"git.vepay.dev/knoknok/backend-platform/pkg/db"
	"git.vepay.dev/knoknok/backend-platform/pkg/di"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

var (
	dbComponent = NewComponent("postgres", initPostgresClient, runPostgresClient)
)

// WithDB добавляет компонент базы данных в сервис (Postgres)
func WithDB() Option {
	return func(app *Application) error {
		app.components.addFirst(component(dbComponent))
		return nil
	}
}

func initPostgresClient(ctx context.Context, app *Application) error {
	logger.Info(ctx, "Postgres initialize")
	config := db.LoadConfig(app.Env)
	manager, err := db.NewPostgresManager(ctx, config)
	if err != nil {
		return err
	}

	app.DB = manager
	di.Register(ctx, app.DB)
	return nil
}

func runPostgresClient(ctx context.Context, app *Application) error {
	if app.DB == nil {
		return fmt.Errorf("db.Client not initialized")
	}

	manager, ok := app.DB.(db.Manager)
	if !ok {
		return fmt.Errorf("failed to cast app.DB to db.Manager")
	}

	if err := manager.Connect(ctx); err != nil {
		return err
	}

	applied, err := manager.Migrate(ctx)
	if err != nil {
		return err
	}
	logger.Info(ctx, "Migrations applied successfully", logger.Int("applied", applied))

	app.Closer.Add(manager.Close)
	app.Health.Add("postgres", func(ctx context.Context) error {
		_, err := manager.HealthStatus()
		return err
	})
	return nil
}
