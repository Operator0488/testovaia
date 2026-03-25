package application

import (
	"context"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/config"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/di"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
)

var containerComponent = NewComponent("di", initContainer, runContainer)

// initContainerClient создает глобальный контейнер
func initContainer(ctx context.Context, app *Application) error {
	container := di.New()
	di.SetGlobal(container)
	app.container = container
	logger.Info(ctx, "DI container created")

	di.RegisterFactory(ctx, func() config.Configurer { return app.Env })
	return nil
}

// runContainer запускается последним
// регистрирует все доступные компоненты Application
func runContainer(ctx context.Context, app *Application) error {
	return app.container.Build()
}
