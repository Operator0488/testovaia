package application

import (
	"context"
	"fmt"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/db"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/di"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
)

var dbComponent = NewComponent("postgres", initPostgresClient, runPostgresClient)

// WithDB добавляет компонент базы данных в сервис (Postgres)
func WithDB() Option {
	return func(app *Application) error {
		app.components.add(component(dbComponent))
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

	app.DB = manager.DB()
	di.Register[db.DbClient](ctx, manager.DB()) // полный клиент
	return nil
}

func runPostgresClient(ctx context.Context, app *Application) error {
	if app.DB == nil {
		return fmt.Errorf("app.DB not initialized")
	}

	manager, ok := app.DB.(db.Manager)
	if !ok {
		return fmt.Errorf("app.DB does not implement db.Manager")
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
