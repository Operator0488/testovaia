package consul

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPairsToObject(t *testing.T) {
	data := []byte(`[
    {
        "Key": "shared/kafka/brokers",
        "CreateIndex": 3285,
        "ModifyIndex": 3285,
        "LockIndex": 0,
        "Flags": 0,
        "Value": "bG9jYWxob3N0OjkwOTIsbG9jYWxob3N0OjkwOTM=",
        "Session": ""
    },
    {
        "Key": "shared/s3/host",
        "CreateIndex": 3280,
        "ModifyIndex": 3280,
        "LockIndex": 0,
        "Flags": 0,
        "Value": "bG9jYWxob3N0",
        "Session": ""
    },
    {
        "Key": "shared/s3/port",
        "CreateIndex": 3281,
        "ModifyIndex": 3281,
        "LockIndex": 0,
        "Flags": 0,
        "Value": "OTA5Mg==",
        "Session": ""
    }
]`)
	pairs := api.KVPairs{}
	err := json.Unmarshal(data, &pairs)
	require.NoError(t, err)
	res, err := convertPairsToObject("shared", pairs)
	require.NoError(t, err)
	assert.Equal(t, "localhost:9092,localhost:9093", getValueByPath(res, "kafka", "brokers"))
	assert.Equal(t, "localhost", getValueByPath(res, "s3", "host"))
	assert.Equal(t, "9092", getValueByPath(res, "s3", "port"))
}

func TestConvertObjectToKeys(t *testing.T) {
	res, err := convertObjectToKeys("shared", map[string]any{
		"kafka": map[string]any{
			"brokers": []string{"localhost:9092", "localhost:9093"},
		},
		"s3": map[string]any{
			"host": "localhost",
			"port": 9092,
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "localhost:9092,localhost:9093", string(res["shared/kafka/brokers"]))
	assert.Equal(t, "localhost", string(res["shared/s3/host"]))
	assert.Equal(t, "9092", string(res["shared/s3/port"]))
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

func getValueByPath(config map[string]any, keys ...string) string {
	if config == nil {
		return "not_exist"
	}
	var value any = config
	for _, key := range keys {
		if keyMap, ok := value.(map[string]any); ok {
			value = keyMap[key]
		}
	}
	if v, ok := value.(string); ok {
		return v
	}
	return "not_exist"
}
