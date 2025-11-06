package application

import (
	"strconv"
	"strings"
	"time"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/config"
	cfg "git.vepay.dev/knoknok/backend-platform/pkg/config"
	"git.vepay.dev/knoknok/backend-platform/pkg/redis"
)

const (
	envKafkaBrokers = "kafka.brokers"
	envKafkaGroup   = "kafka.group"
	envAppName      = cfg.EnvAppName
	envHTTPPort     = "app.port"
	envMetricsAddr  = "metrics.addr"
	envMetricsPort  = "metrics.port"
	envHTTPHost     = "app.host"

	envRedisAddrs        = "redis.addrs"
	envRedisDb           = "redis.db"
	envRedisPoolSize     = "redis.pool_size"
	envRedisDialTimeout  = "redis.dial_timeout"
	envRedisReadTimeout  = "redis.read_timeout"
	envRedisWriteTimeout = "redis.write_timeout"
	envRedisUsername     = "redis.username"
	envRedisPwd          = "redis.pwd"

	defaultKafkaBroker = "kafka:9092"
	defaultMetricsPort = 9091
)

type appConfig struct {
	config.Configurer
}

type kafkaConfig struct {
	Brokers      []string
	DefaultGroup string
}

func (a *appConfig) GetAppName() string {
	return a.GetString(envAppName)
}

// kafka

func (a *appConfig) GetKafkaConfig() kafkaConfig {
	brokers := a.GetStringSlice(envKafkaBrokers)
	if len(brokers) == 0 {
		brokers = []string{defaultKafkaBroker}
	}

	group := getStringOrDefault(a.GetString(envKafkaGroup), a.GetString(envAppName))

	return kafkaConfig{
		Brokers:      brokers,
		DefaultGroup: group,
	}
}

// http

type httpServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func (h *httpServerConfig) GetAddr() string {
	return strings.Join([]string{h.Host, h.Port}, ":")
}

func (a *appConfig) GetHTTPServerConfig() httpServerConfig {
	return httpServerConfig{
		Host:         getStringOrDefault(a.GetString(envHTTPHost), ""),
		Port:         getStringOrDefault(a.GetString(envHTTPPort), "8080"),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// redis

func (a *appConfig) GetRedisConfig() redis.RedisConfig {
	return redis.RedisConfig{
		Addrs:        a.GetStringSlice(envRedisAddrs), // TODO default?
		DB:           a.GetInt(envRedisDb),
		PoolSize:     a.GetInt(envRedisPoolSize),
		DialTimeout:  a.GetDuration(envRedisDialTimeout),
		ReadTimeout:  a.GetDuration(envRedisReadTimeout),
		WriteTimeout: a.GetDuration(envRedisWriteTimeout),
		Username:     a.GetString(envRedisUsername),
		Password:     a.GetString(envRedisPwd),
	}
}

func getStringOrDefault(value string, def string) string {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return def
	}
	return value
}

// metrics

func (a *appConfig) GetMetricsAddr() string {

	if addr := a.GetString(envMetricsAddr); addr != "" {
		return addr
	}

	port := a.GetString(envMetricsPort)
	if port == "" {
		port = strconv.Itoa(defaultMetricsPort)
	}

	if strings.HasPrefix(port, ":") {
		return port
	}

	return ":" + port
}
