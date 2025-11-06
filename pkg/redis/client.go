package redis

import (
	"context"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	"sync"
	"sync/atomic"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	goRedis "github.com/redis/go-redis/v9"
)

const collectMetricsInterval = 5 * time.Second

type Redis interface {
	//базовые команды
	Set(ctx context.Context, key string, val any, ttl time.Duration) error
	Get(ctx context.Context, key string) Value
	Del(ctx context.Context, keys ...string) error

	//для pubsub
	Publish(ctx context.Context, channel string, msg any) error
	Subscribe(ctx context.Context, channels ...string) (Subscriber, error)

	//для healthcheck
	HealthCheck(context.Context) error

	Close() error
}

type client struct {
	cfg   RedisConfig
	codec codec

	cli          atomic.Value // redis UniversalClient
	mu           sync.Mutex
	health       *healthLoop
	lastActivity atomic.Int64 // храним в unix nanos последний успешный запрос

	subsMu sync.RWMutex
	subs   map[*subscriber]struct{}

	poolStop chan struct{} //канал для метрик пула
}

func New(ctx context.Context, cfg RedisConfig) (Redis, error) {

	c := &client{
		cfg:      cfg,
		codec:    JSONCodec{},
		health:   newHealthLoop(),
		subs:     make(map[*subscriber]struct{}),
		poolStop: make(chan struct{}),
	}

	if err := c.rebildClient(ctx, cfg); err != nil {
		logger.Error(ctx, "redis client initialization failed",
			logger.Any("addrs", cfg.Addrs),
			logger.Int("db", cfg.DB),
			logger.String("error", err.Error()),
		)
		return nil, err
	}

	// прогреваем, чтобы были видны в /metrics сразу
	metrics.RedisConnections.WithLabelValues("open").Set(0)
	metrics.RedisConnections.WithLabelValues("idle").Set(0)
	metrics.RedisConnections.WithLabelValues("in_use").Set(0)

	//сбор статы
	go c.collectPoolMetrics()

	go c.health.start(ctx, c)

	logger.Info(ctx, "redis client created",
		logger.Any("addrs", cfg.Addrs),
	)

	return c, nil
}

// пересоздание клиента с новыми кредами
func (c *client) rebildClient(ctx context.Context, cfg RedisConfig) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	opts := &goRedis.UniversalOptions{
		Addrs:        cfg.Addrs,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		Username:     cfg.Username,
		Password:     cfg.Password,
	}

	newcli := goRedis.NewUniversalClient(opts)
	//cli.AddHook(redisLoggerHook{})

	// пингуем новый клиент, если не ок, то закрываем и возвращаем ошибку
	if err := newcli.Ping(ctx).Err(); err != nil {
		_ = newcli.Close()
		logger.Error(ctx, "redis ping failed on rebild",
			logger.Any("addrs", c.cfg.Addrs),
			logger.Int("db", c.cfg.DB),
			logger.String("error", err.Error()),
		)

		return fmt.Errorf("redis ping err, RebilClient: %w", err)
	}

	//сохраняем старые подписки ДО замены клиента
	c.subsMu.RLock()
	activeSubscriptions := c.subs // map[*subscriber]struct{}
	c.subsMu.RUnlock()

	// переподписываем все активные подписки на НОВОМ клиенте TODO: обработка ошибки
	if err := c.resubscribeAll(ctx, activeSubscriptions, newcli); err != nil {
		logger.Error(ctx, "redis resubscribeAll failed on rebild",
			logger.Any("addrs", c.cfg.Addrs),
			logger.Int("db", c.cfg.DB),
			logger.String("error", err.Error()),
		)
	}

	// свап нового клиента
	// старый закрываем
	prev := c.cli.Swap(newcli)
	if prev != nil {
		_ = prev.(goRedis.UniversalClient).Close()
	}

	c.touchActivity()

	logger.Info(ctx, "redis client rebuilt",
		logger.Any("addrs", c.cfg.Addrs),
		logger.Int("db", c.cfg.DB))

	return nil
}

// для доступа к goRedis.UniversalClient
func (c *client) universal() goRedis.UniversalClient {
	return c.cli.Load().(goRedis.UniversalClient)
}

// переподписываем все подписки
func (c *client) resubscribeAll(ctx context.Context, subscriptions map[*subscriber]struct{}, newClient goRedis.UniversalClient) error {

	var errors []error

	// проходим по подписчикам напрямую
	for sub := range subscriptions {
		if err := sub.resubscribe(ctx, newClient); err != nil {
			errors = append(errors, fmt.Errorf("subscriber %p: %w", sub, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to resubscribe %d subscribers: %v",
			len(errors), errors)
	}

	return nil
}

// обновление времени последней успешной активности
func (c *client) touchActivity() {
	c.lastActivity.Store(time.Now().UnixNano())
}

func (c *client) collectPoolMetrics() {
	t := time.NewTicker(collectMetricsInterval)
	defer t.Stop()

	for {
		select {
		case <-c.poolStop:
			return
		case <-t.C:
			cli := c.universal()
			if cli == nil {
				continue
			}
			if ps := cli.PoolStats(); ps != nil {
				open := ps.TotalConns
				idle := ps.IdleConns
				inUse := open - idle
				if inUse < 0 {
					inUse = 0
				}
				metrics.RedisConnections.WithLabelValues("open").Set(float64(open))
				metrics.RedisConnections.WithLabelValues("idle").Set(float64(idle))
				metrics.RedisConnections.WithLabelValues("in_use").Set(float64(inUse))
			}
		}
	}
}

// время последней успешной активности
func (c *client) lastUsed() time.Time {
	return time.Unix(0, c.lastActivity.Load())
}

func (c *client) Close() error {

	select {
	case <-c.poolStop:
		//закрыт
	default:
		close(c.poolStop)
	}

	logger.Info(context.Background(), "redis client closed",
		logger.Any("addrs", c.cfg.Addrs),
		logger.Int("db", c.cfg.DB),
	)

	return c.universal().Close()
}
