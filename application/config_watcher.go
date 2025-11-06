package application

import (
	"context"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

// initConfig запускает слушатель изменения конфига
func (a *Application) initConfig(ctx context.Context) {
	// подписка на изменения
	go a.Env.Watch(ctx)

	a.Closer.Add(func() error {
		err := a.Env.Close(ctx)
		if err == nil {
			logger.Info(ctx, "Config watchers closed successfully")
			return nil
		}

		logger.Error(ctx, "Config watchers closed with error", logger.Err(err))
		return err
	})
}
