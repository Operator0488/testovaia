package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestViperInstance() *viper.Viper {
	viperInst := viper.New()
	viperInst.SetConfigType("yaml")
	viperInst.AddConfigPath("./data")
	viperInst.SetConfigName("viper")
	return viperInst
}

func TestViperConfig(t *testing.T) {
	viperInst := getTestViperInstance()
	err := viperInst.ReadInConfig()

	require.NoError(t, err, "failed to read config")

	assert.Equal(t, "value", viperInst.Get("test"))
	assert.Equal(t, 8080, viperInst.GetInt("app.port"))
	assert.Equal(t, "localhost", viperInst.GetString("app.host"))
	assert.Equal(t, "kafka:9092", viperInst.GetStringSlice("brokers")[0])
	assert.Equal(t, "kafka:9093", viperInst.GetStringSlice("brokers")[1])
}

func TestViperKeys(t *testing.T) {
	viperInst := getTestViperInstance()
	err := viperInst.ReadInConfig()

	require.NoError(t, err, "failed to read config")

	config := viperInst.AllSettings()

	assert.Equal(t, "value", config["test"])
	assert.Equal(t, "8080", config["app"].(map[string]any)["port"])
}

func TestViperMerge(t *testing.T) {
	viperInst := getTestViperInstance()
	err := viperInst.ReadInConfig()

	require.NoError(t, err, "failed to read config")
	configData := map[string]any{
		"s3": map[string]any{
			"auth": map[string]any{
				"pass": "newpass",
			},
		},
		"brokers": []string{"host:9092", "host:9093"},
	}
	viperInst.MergeConfigMap(configData)

	assert.Equal(t, "value", viperInst.Get("test"))
	assert.Equal(t, 8080, viperInst.GetInt("app.port"))
	assert.Equal(t, "localhost", viperInst.GetString("app.host"))
	assert.Equal(t, "host:9092", viperInst.GetStringSlice("brokers")[0])
	assert.Equal(t, "host:9093", viperInst.GetStringSlice("brokers")[1])

	assert.Equal(t, "newpass", viperInst.GetString("s3.auth.pass"))
}
