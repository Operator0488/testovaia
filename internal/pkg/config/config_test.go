package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configprovider "git.vepay.dev/knoknok/backend-platform/internal/pkg/config_provider"
	mock "git.vepay.dev/knoknok/backend-platform/internal/pkg/config_provider/mock"
	gomock "go.uber.org/mock/gomock"
)

func TestConfigLoadEnv(t *testing.T) {
	config := New("./data", "config")
	ctx := context.Background()
	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")

	assert.Equal(t, "value", config.GetString("test"))
	assert.Equal(t, 8080, config.GetInt("app.port"))
	assert.Equal(t, "localhost", config.GetString("app.host"))
	assert.Equal(t, "kafka:9092", config.GetStringSlice("brokers")[0])
	assert.Equal(t, "kafka:9093", config.GetStringSlice("brokers")[1])
}

func TestConfigBootstrap(t *testing.T) {
	config := New("./data", "config_bootstrap")
	ctx := context.Background()

	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")

	ctrl := gomock.NewController(t)
	provider := mock.NewMockProvider(ctrl)

	providerData := map[string]any{
		"kafka": map[string]any{
			"brokers": []string{"host:9092", "host:9093"},
		},
		"service": map[string]any{
			"rate_limit": 101,
		},
	}
	provider.EXPECT().Get(ctx).Return(providerData, nil)
	provider.EXPECT().Set(ctx, map[string]any{
		"app": map[string]any{
			"host": "localhost",
			"port": "8080",
		},
		"kafka": map[string]any{
			"brokers":   []string{"host:9092", "host:9093"},
			"partition": 5,
		},
		"service": map[string]any{
			"rate_limit":   101,
			"duration_min": 60,
		},
	}).Return(nil)

	err = config.Bootstrap(ctx, provider)
	require.NoError(t, err, "failed to bootstrap config")
}

func TestConfigLoadFromProvider(t *testing.T) {
	config := New("./data", "config_provider")
	ctx := context.Background()

	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")

	ctrl := gomock.NewController(t)
	provider := mock.NewMockProvider(ctrl)

	providerData := map[string]any{
		"kafka": map[string]any{
			"brokers": []string{"host:9092", "host:9093"},
		},
		"service": map[string]any{
			"rate_limit": 101,
		},
	}
	provider.EXPECT().Get(ctx).Return(providerData, nil)

	err = config.LoadFromProvider(ctx, provider)
	require.NoError(t, err, "failed to load config from provider")

	assert.Equal(t, []string{"host:9092", "host:9093"}, config.GetStringSlice("kafka.brokers"))
	assert.Equal(t, 101, config.GetInt("service.rate_limit"))
	assert.Equal(t, 8080, config.GetInt("app.port"))
}

func TestConfigLoadFromProviderWithPriority(t *testing.T) {
	config := New("./data", "config_provider")
	ctx := context.Background()

	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")

	ctrl := gomock.NewController(t)

	// provider 1
	provider1 := mock.NewMockProvider(ctrl)
	providerData1 := map[string]any{
		"kafka": map[string]any{
			"brokers": []string{"host:9092", "host:9093"},
		},
		"service": map[string]any{
			"rate_limit": 101,
		},
	}
	provider1.EXPECT().Get(ctx).Return(providerData1, nil)

	// provider 2
	provider2 := mock.NewMockProvider(ctrl)
	providerData2 := map[string]any{
		"kafka": map[string]any{
			"brokers": []string{"provider:9092", "provider:9093"},
		},
		"service": map[string]any{
			"rate_limit": 101,
		},
	}
	provider2.EXPECT().Get(ctx).Return(providerData2, nil)

	err = config.LoadFromProvider(ctx, provider1)
	require.NoError(t, err, "failed to load config from provider 1")

	err = config.LoadFromProvider(ctx, provider2)
	require.NoError(t, err, "failed to load config from provider 2 ")

	assert.Equal(t, []string{"provider:9092", "provider:9093"}, config.GetStringSlice("kafka.brokers")) // provider 2
	assert.Equal(t, 101, config.GetInt("service.rate_limit"))                                           // provider 1
	assert.Equal(t, 8080, config.GetInt("app.port"))                                                    // config env
}

func TestConfigLoadFromProviderWithWatcher(t *testing.T) {
	config := New("./data", "config_provider")
	ctx := context.Background()

	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")

	provider := &mockWatchingProvider{}

	err = config.LoadFromProvider(ctx, provider)
	require.NoError(t, err, "failed to load config from provider")

	// phase 1, not modified
	assert.Equal(t, []string{"kafka:9092", "kafka:9093"}, config.GetStringSlice("kafka.brokers"))
	assert.Equal(t, 100, config.GetInt("service.rate_limit"))
	assert.Equal(t, 8080, config.GetInt("app.port"))

	// start watching
	config.Watch(ctx)

	// send changes
	changes := map[string]any{
		"kafka": map[string]any{
			"brokers": []string{"provider:9092", "provider:9093"},
		},
		"service": map[string]any{
			"rate_limit": 101,
		},
	}
	provider.Callback(changes)

	// phase 2, check updates
	assert.Equal(t, []string{"provider:9092", "provider:9093"}, config.GetStringSlice("kafka.brokers")) // changed
	assert.Equal(t, 101, config.GetInt("service.rate_limit"))                                           // changed
	assert.Equal(t, 8080, config.GetInt("app.port"))

	// phase 3, check channel
	value := <-config.watchChan
	assert.Equal(t, changes, value)
}

func TestConfigLoadFromProviderWithSubscriber(t *testing.T) {
	config := New("./data", "config_provider")
	ctx := context.Background()

	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")

	provider := &mockWatchingProvider{}

	err = config.LoadFromProvider(ctx, provider)
	require.NoError(t, err, "failed to load config from provider")

	// phase 1, not modified
	assert.Equal(t, []string{"kafka:9092", "kafka:9093"}, config.GetStringSlice("kafka.brokers"))
	assert.Equal(t, 100, config.GetInt("service.rate_limit"))
	assert.Equal(t, 8080, config.GetInt("app.port"))

	// start watching
	config.Watch(ctx)

	configWatcher := NewConfigWatcher("test", config, newComponentConfig)
	config.Subscribe(configWatcher)
	assert.Equal(t, 100, configWatcher.Get().ServiceLimit)

	// send changes
	changes := map[string]any{
		"kafka": map[string]any{
			"brokers": []string{"provider:9092", "provider:9093"},
		},
		"service": map[string]any{
			"rate_limit": 101,
		},
	}
	provider.Callback(changes)

	// phase 2, check updates
	assert.Equal(t, []string{"provider:9092", "provider:9093"}, config.GetStringSlice("kafka.brokers")) // changed
	assert.Equal(t, 101, config.GetInt("service.rate_limit"))                                           // changed
	assert.Equal(t, 8080, config.GetInt("app.port"))

	// phase 3, check channel
	config.triggerUpdates(ctx)
	assert.Equal(t, 101, configWatcher.Get().ServiceLimit)
}

type mockComponentConfig struct {
	ServiceLimit int
}

func newComponentConfig(cfg Configurer) mockComponentConfig {
	return mockComponentConfig{ServiceLimit: cfg.GetInt("service.rate_limit")}
}

type mockWatchingProvider struct {
	Callback func(data map[string]interface{})
}

func (c *mockWatchingProvider) Close(ctx context.Context) error {
	return nil
}

func (c *mockWatchingProvider) Get(ctx context.Context) (configprovider.ConfigData, error) {
	return nil, nil
}

func (c *mockWatchingProvider) Set(ctx context.Context, value configprovider.ConfigData) error {
	return nil
}

func (c *mockWatchingProvider) Watch(ctx context.Context, onChange func(map[string]interface{})) error {
	c.Callback = onChange
	return nil
}
