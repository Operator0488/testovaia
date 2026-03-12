package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigWatcher(t *testing.T) {
	config := New("./data", "config_provider")
	ctx := context.Background()

	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")
	durationMin1 := 61
	durationMin2 := 62
	version := 1
	watcher := NewConfigWatcher(
		"test",
		config,
		func(c Configurer) testConfig {
			if version == 1 {
				return testConfig{
					DurationMin: durationMin1,
				}
			}
			return testConfig{
				DurationMin: durationMin2,
			}
		})
	version = 2

	handleRefresh := false
	watcher.OnRefresh(func(cfg testConfig) error {
		handleRefresh = true
		return nil
	})

	ok, err := watcher.TryUpdate()
	require.NoError(t, err, "failed to update config")

	assert.Equal(t, true, ok)
	assert.Equal(t, 62, watcher.Get().DurationMin)
	assert.Equal(t, true, handleRefresh)
	assert.Equal(t, "test", watcher.GetName())
}

func TestNewConfigWatcherNotChanged(t *testing.T) {
	config := New("./data", "config_provider")
	ctx := context.Background()

	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")
	watcher := NewConfigWatcher(
		"test",
		config,
		func(c Configurer) *testConfig2 {
			return &testConfig2{
				Endpoint:        "localhost",
				AccessKeyID:     "admin",
				SecretAccessKey: "secret",
				Bucket:          "bucket",
				Region:          "region",
				UseSSL:          false,
				CreateBucket:    true,
			}
		})

	ok, err := watcher.TryUpdate()
	require.NoError(t, err, "failed to update config")

	assert.Equal(t, false, ok)
	assert.Equal(t, "test", watcher.GetName())
}

func TestNewConfigWatcherWithoutCallback(t *testing.T) {
	config := New("./data", "config_provider")
	ctx := context.Background()

	err := config.LoadEnv(ctx)
	require.NoError(t, err, "failed to read config")

	durationMin1 := 61
	durationMin2 := 62
	version := 1
	watcher := NewConfigWatcher(
		"test",
		config,
		func(c Configurer) testConfig {
			if version == 1 {
				return testConfig{
					DurationMin: durationMin1,
				}
			}
			return testConfig{
				DurationMin: durationMin2,
			}
		})
	version = 2

	ok, err := watcher.TryUpdate()
	require.NoError(t, err, "failed to update config")

	assert.Equal(t, 62, watcher.Get().DurationMin)
	assert.Equal(t, "test", watcher.GetName())
	assert.Equal(t, true, ok)
}

type testConfig struct {
	DurationMin int
}

type testConfig2 struct {
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `json:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key" yaml:"secret_access_key"`
	Bucket          string `json:"bucket" yaml:"bucket"`
	Region          string `json:"region" yaml:"region"`
	UseSSL          bool   `json:"use_ssl" yaml:"use_ssl"`
	CreateBucket    bool   `json:"create_bucket" yaml:"create_bucket"`
}
