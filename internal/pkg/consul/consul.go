package consul

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

var (
	ErrNotFound = errors.New("value by key not found")
)

type Client interface {
	GetConfig(key string) (api.KVPairs, error)
	Insert(key string, value []byte) error
	WatchPrefix(ctx context.Context, prefix string, callback func(api.KVPairs)) error
	Close() error
}

type consulClient struct {
	client *api.Client
	kv     *api.KV
	plans  []*watch.Plan
	mu     *sync.Mutex
}

// NewClient create client for Consul
func NewClient(address string) (Client, error) {
	config := api.DefaultConfig()
	config.Address = address
	return NewClientWithConfig(config)
}

// NewClientWithConfig create client by config.
func NewClientWithConfig(config *api.Config) (Client, error) {
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	return &consulClient{
		client: client,
		kv:     client.KV(),
		mu:     &sync.Mutex{},
	}, nil
}

// GetConfig config pairs by prefix
func (c *consulClient) GetConfig(prefix string) (api.KVPairs, error) {
	pairs, _, err := c.kv.List(prefix, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get config from consul: %w", err)
	}

	return pairs, err
}

// Insert create new config by key
func (c *consulClient) Insert(key string, value []byte) error {
	pair := &api.KVPair{
		Key:         key,
		Value:       value,
		ModifyIndex: 0, // create if not exist
	}

	_, _, err := c.kv.CAS(pair, nil)
	if err != nil {
		return fmt.Errorf("failed to create consul config: %w", err)
	}

	return nil
}

// WatchPrefix start watcher by prefix
func (c *consulClient) WatchPrefix(ctx context.Context, prefix string, callback func(api.KVPairs)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	plan, err := watch.Parse(map[string]interface{}{
		"type":   "keyprefix",
		"prefix": prefix,
	})
	if err != nil {
		return fmt.Errorf("failed to parse watch plan: %w", err)
	}

	plan.Handler = func(idx uint64, data interface{}) {
		if data == nil {
			return
		}

		pairs, ok := data.(api.KVPairs)
		if !ok {
			logger.Error(ctx, "Unexpected data type in prefix watch",
				logger.Err(err),
				logger.String("prefix", prefix),
			)
			return
		}

		callback(pairs)
	}

	c.plans = append(c.plans, plan)

	go func() {
		if err := plan.RunWithClientAndHclog(c.client, nil); err != nil {
			logger.Error(ctx, "Consul watch prefix plan failed",
				logger.Err(err),
				logger.String("prefix", prefix),
			)
		}
	}()

	return nil
}

func (c *consulClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, p := range c.plans {
		p.Stop()
	}
	return nil
}
