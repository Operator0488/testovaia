package etcd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToObject(t *testing.T) {
	pairs := map[string]string{
		"shared/kafka/brokers": "localhost:9092,localhost:9093",
		"shared/s3/host":       "localhost",
		"shared/s3/port":       "9092",
	}

	res := convertToObject("shared", pairs)

	assert.Equal(t, "localhost:9092,localhost:9093", getValueByPath(res, "kafka", "brokers"))
	assert.Equal(t, "localhost", getValueByPath(res, "s3", "host"))
	assert.Equal(t, "9092", getValueByPath(res, "s3", "port"))
}

func TestConvertToObject_EmptyMap(t *testing.T) {
	res := convertToObject("shared", map[string]string{})
	assert.Empty(t, res)
}

func TestConvertToObject_PrefixOnlyKey(t *testing.T) {
	pairs := map[string]string{
		"shared/": "",
	}
	res := convertToObject("shared", pairs)
	assert.Empty(t, res)
}

func TestConvertObjectToKeys(t *testing.T) {
	res, err := convertObjectToKeys("shared", map[string]any{
		"s3": map[string]any{
			"host": "localhost",
			"port": 9092,
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "localhost", res["shared/s3/host"])
	assert.Equal(t, "9092", res["shared/s3/port"])
}

func TestConvertObjectToKeys_NilInput(t *testing.T) {
	res, err := convertObjectToKeys("shared", nil)
	require.NoError(t, err)
	assert.Nil(t, res)
}

func TestAddToObject(t *testing.T) {
	config := make(map[string]any)
	addToObject(config, "s3/host", "abc")
	addToObject(config, "s3", "abc")
	assert.Equal(t, "abc", config["s3"].(map[string]any)["host"])
}

func TestAddToObject2(t *testing.T) {
	config := make(map[string]any)
	addToObject(config, "s3", "abc")
	addToObject(config, "s3/host", "abc")
	assert.Equal(t, "abc", config["s3"].(map[string]any)["host"])
}

func TestRoundTrip(t *testing.T) {
	original := map[string]any{
		"db": map[string]any{
			"host": "localhost",
			"port": "5432",
		},
		"kafka": map[string]any{
			"topic": "events",
		},
	}

	flat, err := convertObjectToKeys("app", original)
	require.NoError(t, err)

	restored := convertToObject("app", flat)
	assert.Equal(t, original, restored)
}

