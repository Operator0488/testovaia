package consul

import (
	"context"
	"fmt"

	configprovider "git.vepay.dev/knoknok/backend-platform/internal/pkg/config_provider"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/consul"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"github.com/hashicorp/consul/api"
	"golang.org/x/sync/errgroup"
)

type consulProvider struct {
	prefix string
	client consul.Client
}

func NewProvider(prefix string, client consul.Client) configprovider.Provider {
	return &consulProvider{
		client: client,
		prefix: prefix,
	}
}

func (c *consulProvider) Close(ctx context.Context) error {
	return c.client.Close()
}

func (c *consulProvider) Get(ctx context.Context) (configprovider.ConfigData, error) {
	pairs, err := c.client.GetConfig(c.prefix)
	if err != nil {
		return nil, err
	}
	return convertPairsToObject(c.prefix, pairs)
}

func (c *consulProvider) Set(ctx context.Context, value configprovider.ConfigData) error {
	data, err := convertObjectToKeys(c.prefix, value)
	if err != nil {
		return err
	}

	group, _ := errgroup.WithContext(ctx)

	// concurrently save all keys
	for key, value := range data {
		group.Go(func() error {
			if err := c.client.Insert(key, value); err != nil {
				return &SaveError{
					Key: key,
					Err: err,
				}
			}
			return nil
		})
	}

	return group.Wait()
}

func (c *consulProvider) Watch(ctx context.Context, onChange func(map[string]any)) error {
	return c.client.WatchPrefix(ctx, c.prefix, func(pairs api.KVPairs) {
		config, err := convertPairsToObject(c.prefix, pairs)
		if err != nil {
			logger.Error(ctx, "failed to process consul watcher key value pairs", logger.Err(err))
			return
		}
		onChange(config)
	})
}

type SaveError struct {
	Key string
	Err error
}

func (e *SaveError) Error() string {
	return fmt.Sprintf("save key '%s' failed: %v", e.Key, e.Err)
}

func (e *SaveError) Unwrap() error {
	return e.Err
}
