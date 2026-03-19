package etcd

import (
	"context"
	"fmt"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Client interface {
	GetByPrefix(ctx context.Context, prefix string) (map[string]string, int64, error)
	Put(ctx context.Context, key string, value string) error
	WatchPrefix(ctx context.Context, prefix string, rev int64, callback func(map[string]string)) error
	Close() error
}

type etcdClient struct {
	client *clientv3.Client
}

func NewClientWithConfig(config clientv3.Config) (Client, error) {
	client, err := clientv3.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &etcdClient{client: client}, nil
}

func (c *etcdClient) GetByPrefix(ctx context.Context, prefix string) (map[string]string, int64, error) {
	resp, err := c.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get config from etcd: %w", err)
	}

	result := make(map[string]string, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = string(kv.Value)
	}
	return result, resp.Header.Revision, nil
}

func (c *etcdClient) Put(ctx context.Context, key string, value string) error {
	_, err := c.client.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("failed to put config to etcd: %w", err)
	}
	return nil
}

func (c *etcdClient) WatchPrefix(ctx context.Context, prefix string, rev int64, callback func(map[string]string)) error {
	opts := []clientv3.OpOption{clientv3.WithPrefix()}
	if rev > 0 {
		opts = append(opts, clientv3.WithRev(rev))
	}

	resp, err := c.client.Get(ctx, prefix, opts...)
	if err != nil {
		return fmt.Errorf("failed to get initial state for watch: %w", err)
	}

	state := make(map[string]string, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		state[string(kv.Key)] = string(kv.Value)
	}
	watchRev := resp.Header.Revision + 1

	go func() {
		wch := c.client.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(watchRev))
		for wresp := range wch {
			if wresp.Err() != nil {
				logger.Error(ctx, "etcd watch error",
					logger.Err(wresp.Err()),
					logger.String("prefix", prefix),
				)
				return
			}

			for _, ev := range wresp.Events {
				key := string(ev.Kv.Key)
				switch ev.Type {
				case mvccpb.PUT:
					state[key] = string(ev.Kv.Value)
				case mvccpb.DELETE:
					delete(state, key)
				}
			}

			snapshot := make(map[string]string, len(state))
			for k, v := range state {
				snapshot[k] = v
			}
			callback(snapshot)
		}
	}()

	return nil
}

func (c *etcdClient) Close() error {
	return c.client.Close()
}
