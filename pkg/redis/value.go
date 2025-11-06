package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	goRedis "github.com/redis/go-redis/v9"
	"time"
)

type Value struct {
	data string
	err  error
}

func (v Value) Scan(dst any) error {
	if v.err != nil {
		return v.err
	}
	return json.Unmarshal([]byte(v.data), dst)
}

func (v Value) Err() error {
	if v.err != nil {
		return v.err
	}
	return nil
}

func (v Value) IsNotFound() bool {
	return errors.Is(v.err, ErrNotFound)
}

func (c *client) Get(ctx context.Context, key string) Value {

	start := time.Now()
	val, err := c.universal().Get(ctx, key).Result()
	metrics.RedisQueryDuration.WithLabelValues("GET").Observe(time.Since(start).Seconds())

	if errors.Is(err, goRedis.Nil) {
		return Value{"", ErrNotFound}
	}

	if err == nil {
		c.touchActivity()
	}

	if err != nil {
		logger.Error(ctx, "redis get failed",
			logger.String("key", key),
			logger.String("error", err.Error()),
		)
	}

	return Value{val, err}
}

func (c *client) Set(ctx context.Context, key string, val any, ttl time.Duration) error {

	bt, err := c.codec.Marshal(val)
	if err != nil {
		return err
	}

	start := time.Now()
	err = c.universal().Set(ctx, key, bt, ttl).Err()
	metrics.RedisQueryDuration.WithLabelValues("SET").Observe(time.Since(start).Seconds())

	if err == nil {
		c.touchActivity()
		return nil
	}

	logger.Error(ctx, "redis set failed",
		logger.String("key", key),
		logger.String("error", err.Error()),
	)

	return fmt.Errorf("redis set key=%q: %w", key, err)
}

func (c *client) Del(ctx context.Context, keys ...string) error {

	start := time.Now()
	err := c.universal().Del(ctx, keys...).Err()
	metrics.RedisQueryDuration.WithLabelValues("DEL").Observe(time.Since(start).Seconds())

	if err == nil {
		c.touchActivity()
		return nil
	}

	logger.Error(ctx, "redis del failed",
		logger.Any("keys", keys),
		logger.String("error", err.Error()),
	)

	return fmt.Errorf("redis del keys=%q: %w", keys, err)
}

func (c *client) Publish(ctx context.Context, channel string, msg any) error {

	b, err := c.codec.Marshal(msg)
	if err != nil {
		logger.Error(ctx, "redis marshal failed on publish",
			logger.String("channel", channel),
			logger.String("error", err.Error()),
		)
		return err
	}

	start := time.Now()
	err = c.universal().Publish(ctx, channel, b).Err()
	metrics.RedisQueryDuration.WithLabelValues("PUBLISH").Observe(time.Since(start).Seconds())

	if err == nil {
		c.touchActivity()
		return nil
	}

	logger.Error(ctx, "redis publish failed",
		logger.String("channel", channel),
		logger.Any("msg", msg),
		logger.String("error", err.Error()),
	)

	return fmt.Errorf("redis publish channel=%v, msg=%v: %w", channel, msg, err)

}
