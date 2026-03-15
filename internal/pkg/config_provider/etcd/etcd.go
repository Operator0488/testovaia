package etcd

import (
	"context"
	"fmt"

	configprovider "easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/config_provider"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/etcd"
	"golang.org/x/sync/errgroup"
)

type etcdProvider struct {
	client       etcd.Client
	prefix       string
	lastRevision int64
}

func (e *etcdProvider) Get(ctx context.Context) (configprovider.ConfigData, error) {
	pairs, rev, err := e.client.GetByPrefix(ctx, e.prefix)
	if err != nil {
		return nil, err
	}
	e.lastRevision = rev
	return convertToObject(e.prefix, pairs), nil
}

func (e *etcdProvider) Set(ctx context.Context, value configprovider.ConfigData) error {
	newData, err := convertObjectToKeys(e.prefix, value)
	if err != nil {
		return err
	}

	currentData, _, err := e.client.GetByPrefix(ctx, e.prefix)
	if err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(ctx)

	for key, val := range newData {
		if _, exists := currentData[key]; exists {
			continue
		}
		group.Go(func() error {
			if err := e.client.Put(groupCtx, key, val); err != nil {
				return fmt.Errorf("save key '%s' failed: %w", key, err)
			}
			return nil
		})
	}

	return group.Wait()
}

func (e *etcdProvider) Watch(ctx context.Context, onChange func(map[string]interface{})) error {
	return e.client.WatchPrefix(ctx, e.prefix, e.lastRevision, func(pairs map[string]string) {
		config := convertToObject(e.prefix, pairs)
		onChange(config)
	})
}

func (e *etcdProvider) Close(ctx context.Context) error {
	return e.client.Close()
}

func NewProvider(client etcd.Client, prefix string) configprovider.Provider {
	return &etcdProvider{
		client: client,
		prefix: prefix,
	}
}
