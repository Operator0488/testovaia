package config

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

type ConfigWatcher[T any] struct {
	configData T
	onRefresh  func(cfg T) error
	create     func(Configurer) T
	cfg        Configurer
	mu         *sync.RWMutex
	name       string
}

type IConfigSubscriber interface {
	TryUpdate() (bool, error)
	GetName() string
}

type IConfigWatcher[T any] interface {
	Get() T
	GetName() string
	TryUpdate() (bool, error)
	OnRefresh(cb func(cfg T) error)
}

type comparable[T any] interface {
	Compare(other T) bool
}

var _ IConfigSubscriber = (*ConfigWatcher[any])(nil)
var _ IConfigWatcher[any] = (*ConfigWatcher[any])(nil)

// NewConfigWatcher create config wrapper, which safely update config.
func NewConfigWatcher[T any](name string, cfg Configurer, create func(Configurer) T) *ConfigWatcher[T] {
	return &ConfigWatcher[T]{
		configData: create(cfg),
		create:     create,
		cfg:        cfg,
		mu:         &sync.RWMutex{},
		name:       name,
	}
}

func (c *ConfigWatcher[T]) Get() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configData
}

func (c *ConfigWatcher[T]) OnRefresh(cb func(cfg T) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onRefresh = cb
}

func (c *ConfigWatcher[T]) TryUpdate() (ok bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ok = true
	newCfg := c.create(c.cfg)

	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic unhandled error")
			logger.Error(context.Background(),
				"Panic in config watcher.TryUpdate method",
				logger.String("component", c.GetName()),
			)
		}
	}()

	if c.equalConfig(newCfg, c.configData) {
		ok = false
		return
	}

	if c.onRefresh != nil {
		if err = c.onRefresh(newCfg); err != nil {
			ok = false
			return
		}
	}
	c.configData = newCfg
	return
}

func (c *ConfigWatcher[T]) GetName() string {
	return c.name
}

// equalConfig
func (c *ConfigWatcher[T]) equalConfig(a, b T) bool {
	if comparable, ok := any(a).(comparable[T]); ok {
		return comparable.Compare(b)
	}

	return reflect.DeepEqual(a, b)
}
