package config

import "easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/config"

type IConfigWatcher[T any] = config.IConfigWatcher[T]

type IConfigSubscriber = config.IConfigSubscriber

type Configurer = config.Configurer

// NewConfigWatcher create config wrapper, which safely update config.
func NewConfigWatcher[T any](name string, cfg config.Configurer, create func(config.Configurer) T) IConfigWatcher[T] {
	return config.NewConfigWatcher(name, cfg, create)
}
