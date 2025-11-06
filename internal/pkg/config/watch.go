package config

import (
	"context"
	"errors"
	"strings"
	"sync"

	configprovider "git.vepay.dev/knoknok/backend-platform/internal/pkg/config_provider"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

// Watch run watching changes from config server providers.
func (c *Config) Watch(ctx context.Context) {
	c.watchChan = make(chan map[string]any, 1)

	for _, storage := range c.storages {
		if err := c.watchStorage(ctx, storage); err != nil {
			logger.Error(ctx,
				"config provider watching failed",
				logger.Err(err),
			)
		}
	}

	go func() {
		for config := range c.watchChan {
			keys := strings.Builder{}
			for key := range config {
				keys.WriteString(key)
				keys.WriteString(", ")
			}
			logger.Info(ctx,
				"Config server updating",
				logger.String("keys", keys.String()),
			)
			c.triggerUpdates(ctx)
		}
	}()
}

// Subscribe add subscribers to change config.
func (c *Config) Subscribe(s IConfigSubscriber) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscribes = append(c.subscribes, s)
}

func (c *Config) watchStorage(ctx context.Context, storage *storage) error {
	err := storage.provider.Watch(ctx, func(data map[string]any) {
		c.mu.Lock()
		defer c.mu.Unlock()

		if c.watchChan == nil || c.closed {
			return
		}

		if err := storage.viper.MergeConfigMap(data); err != nil {
			logger.Error(ctx,
				"failed to update config from provider watcher",
				logger.Err(err),
			)
			return
		}

		select {
		case c.watchChan <- data:
		default:
		}
	})

	if err != nil {
		if errors.Is(err, configprovider.ErrUnsupported) {
			return nil
		}
	}
	return err
}

func (c *Config) triggerUpdates(ctx context.Context) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	wg := sync.WaitGroup{}
	wg.Add(len(c.subscribes))
	for _, subscriber := range c.subscribes {
		go func() {
			defer wg.Done()

			ok, err := subscriber.TryUpdate()
			if err != nil {
				logger.Error(ctx,
					"failed update config for component",
					logger.String("component", subscriber.GetName()),
					logger.Err(err),
				)
				return
			}
			if ok {
				logger.Info(ctx,
					"component reinitilized successfully",
					logger.String("component", subscriber.GetName()),
				)
			}
		}()
	}

	wg.Wait()
}
