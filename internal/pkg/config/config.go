package config

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	cp "git.vepay.dev/knoknok/backend-platform/internal/pkg/config_provider"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"github.com/spf13/viper"
)

var ErrUnsupported = errors.New("unsupported operation")

type storage struct {
	provider cp.Provider
	viper    *viper.Viper
}

type Config struct {
	mu         *sync.RWMutex // concurrency when watching from provider
	configPath string
	fileName   string
	envViper   *viper.Viper
	storages   []*storage
	subscribes []IConfigSubscriber
	watchChan  chan map[string]interface{}
	closed     bool
}

//go:generate go run go.uber.org/mock/mockgen -destination=mock/mock.go -package=mock -source=config.go

type Configurer interface {
	GetString(key string) string
	GetBool(key string) bool
	GetBoolOrDefault(key string, def bool) bool
	GetInt(key string) int
	GetInt32(key string) int32
	GetInt64(key string) int64
	GetStringSlice(key string) []string
	GetIntSlice(key string) []int
	GetDuration(key string) time.Duration
	GetStringOrDefault(key string, def string) string
	GetIntOrDefault(key string, def int) int
	GetFloat64(key string) float64

	// Watch run watching changes from config server providers.
	Watch(ctx context.Context)

	// Subscribe add subscribers to change config.
	Subscribe(IConfigSubscriber)

	// Close stop watchers.
	Close(ctx context.Context) error
}

// New создает новый экземпляр конфигурации
func New(configPath string, fileName string) *Config {
	return &Config{
		configPath: configPath,
		fileName:   fileName,
		mu:         &sync.RWMutex{},
	}
}

// LoadEnv load config from file, env
func (c *Config) LoadEnv(ctx context.Context) error {
	viperInst := viper.New()
	viperInst.SetConfigType("yaml")
	viperInst.AddConfigPath(c.configPath)
	viperInst.AddConfigPath(".")
	viperInst.AddConfigPath("./configs")
	viperInst.SetConfigName(c.fileName)
	viperInst.AutomaticEnv()

	// Load from file
	if err := viperInst.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load config from file %w", err)
	}

	c.envViper = viperInst
	return nil
}

// Bootstrap config file to provider.
// Load config file envs and save to config provider
func (c *Config) Bootstrap(ctx context.Context, provider cp.Provider) error {
	viperInst := viper.New()
	viperInst.SetConfigType("yaml")
	viperInst.AddConfigPath(c.configPath)
	viperInst.AddConfigPath(".")
	viperInst.SetConfigName(c.fileName)

	// Load from file
	if err := viperInst.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load config from file %w", err)
	}

	providerData, err := provider.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config from provider %w", err)
	}

	if providerData != nil {
		if err := viperInst.MergeConfigMap(providerData); err != nil {
			return fmt.Errorf("failed to merge config from provider %w", err)
		}
	}

	configData := viperInst.AllSettings()

	// Update provider config
	if err := provider.Set(ctx, configData); err != nil {
		if errors.Is(err, cp.ErrUnsupported) {
			logger.Warn(ctx, "provider not supported bootstraping")
			return nil
		}
		return fmt.Errorf("failed to save config to provider %w", err)
	}
	return nil
}

// LoadFromProvider create new storage for provider and register this storage in config
func (c *Config) LoadFromProvider(ctx context.Context, provider cp.Provider) error {
	viperInst := viper.New()
	providerData, err := provider.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config from provider %w", err)
	}

	if providerData != nil {
		if err := viperInst.MergeConfigMap(providerData); err != nil {
			return fmt.Errorf("failed to merge config from provider %w", err)
		}
	}

	c.storages = append([]*storage{{viper: viperInst, provider: provider}}, c.storages...)
	return nil
}
