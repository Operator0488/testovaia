package config

import "git.vepay.dev/knoknok/backend-platform/internal/pkg/config"

type IConfigWatcher[T any] = config.IConfigWatcher[T]

type Configurer = config.Configurer

// NewConfigWatcher create config wrapper, which safely update config.
func NewConfigWatcher[T any](name string, cfg config.Configurer, create func(config.Configurer) T) IConfigWatcher[T] {
	return config.NewConfigWatcher(name, cfg, create)
}
