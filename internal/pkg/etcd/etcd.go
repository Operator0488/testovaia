package etcd

import (
	"context"
	"fmt"
	"time"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Client interface {
	GetByPrefix(ctx context.Context, prefix string) (map[string]string, int64, error)
	Put(ctx context.Context, key string, value string) error
	WatchPrefix(ctx context.Context, prefix string, callback func(map[string]string)) error
	Close() error
}

type etcdClient struct {
	client              *clientv3.Client
	lastKnownRevision int64
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

	c.lastKnownRevision = resp.Header.Revision

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

// WatchPrefix запускает цикл наблюдения за префиксом в etcd.
// При каждом (пере)подключении делает полный snapshot через Get, затем подписывается на изменения.
// При ошибке watch автоматически переподключается с экспоненциальным backoff.
// Завершается только при отмене ctx.
func (c *etcdClient) WatchPrefix(ctx context.Context, prefix string, callback func(map[string]string)) error {
	go func() {
		backoff := time.Second
		seenRevision := c.lastKnownRevision

		for {
			if ctx.Err() != nil {
				return
			}

			// Свежий snapshot
			resp, err := c.client.Get(ctx, prefix, clientv3.WithPrefix())
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Error(ctx, "etcd watch: failed to get snapshot",
					logger.Err(err),
					logger.String("prefix", prefix),
				)
				c.sleepOrDone(ctx, backoff)
				backoff = nextBackoff(backoff)
				continue
			}

			state := make(map[string]string, len(resp.Kvs))
			for _, kv := range resp.Kvs {
				state[string(kv.Key)] = string(kv.Value)
			}

			// Пропускаем callback если ревизия не изменилась с последнего известного состояния
			if seenRevision != resp.Header.Revision {
				callback(copyMap(state))
			}

			seenRevision = resp.Header.Revision

			watchRev := resp.Header.Revision + 1
			backoff = time.Second // сброс backoff после успешного Get

			// Подписываемся на изменения начиная с revision+1
			wch := c.client.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(watchRev))
			for wresp := range wch {
				if wresp.Err() != nil {
					logger.Warn(ctx, "etcd watch error, will reconnect",
						logger.Err(wresp.Err()),
						logger.String("prefix", prefix),
					)
					break
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

				callback(copyMap(state))
			}

			// Если context отменён — выходим без retry
			if ctx.Err() != nil {
				return
			}

			logger.Warn(ctx, "etcd watch interrupted, retrying",
				logger.String("prefix", prefix),
				logger.String("backoff", backoff.String()),
			)
			c.sleepOrDone(ctx, backoff)
			backoff = nextBackoff(backoff)
		}
	}()

	return nil
}

func (c *etcdClient) Close() error {
	return c.client.Close()
}

func copyMap(m map[string]string) map[string]string {
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func nextBackoff(current time.Duration) time.Duration {
	const maxBackoff = 30 * time.Second
	next := current * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}

// sleepOrDone ждёт указанную длительность или отмену контекста.
func (c *etcdClient) sleepOrDone(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
