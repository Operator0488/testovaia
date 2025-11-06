package redis

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	//goRedis "github.com/redis/go-redis/v9"
)

// интервал пинга в секундах
const interval = 30

// структура для хранения состояния здоровья
type Health struct {
	OK      bool
	Latency time.Duration
	LastErr error
}

// внутренняя структура для хранения состояния здоровья
type healthLoop struct {
	ok  atomic.Bool
	lat atomic.Int64
	err atomic.Value
}

func newHealthLoop() *healthLoop {
	return &healthLoop{}
}

func (h *healthLoop) start(ctx context.Context, c *client) {

	// Пинговать редис раз в interval
	// если редис недавно использовали, то скип
	// если пинг успешен, то обновить lastOk и latency
	// если пинг неуспешен, то обновить lastOk и lastErr

	ticker := time.NewTicker(interval * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():
			return

		case <-ticker.C:
			// если редис недавно использовали, то скип

			if time.Since(c.lastUsed()) < interval {
				continue
			}

			start := time.Now()

			if err := c.universal().Ping(ctx).Err(); err != nil {
				h.ok.Store(false)
				h.err.Store(err)
				logger.Warn(ctx, "redis health check failed",
					logger.String("error", err.Error()),
				)
			} else {
				h.ok.Store(true)
				//h.err.Store(error(nil)) (ошибка, nil будет выдавать ошибку в Value)
				h.lat.Store(time.Since(start).Microseconds())
				logger.Debug(ctx, "redis health check ok",
					logger.Duration("latency", time.Since(start)),
				)

			}
		}
	}
}

func (c *client) latency() time.Duration {
	return time.Duration(c.health.lat.Load()) * time.Microsecond
}

func (c *client) snapshot() Health {

	if c.health.ok.Load() {
		return Health{
			OK:      true,
			Latency: c.latency(),
			LastErr: nil,
		}
	}

	val := c.health.err.Load()

	var err error

	if val != nil {
		err = val.(error)
	}

	return Health{
		OK:      c.health.ok.Load(),
		Latency: c.latency(),
		LastErr: err,
	}
}

func (c *client) HealthCheck(_ context.Context) error {

	snap := c.snapshot()
	if snap.OK {
		return nil
	}

	if snap.LastErr != nil {
		return snap.LastErr
	}

	return fmt.Errorf("redis unhealthy, latency=%v", snap.Latency)

}
