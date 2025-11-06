package consul

import (
	"context"
	"fmt"
	"strings"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"github.com/hashicorp/consul/api"
)

const (
	pathSeparator = "/"
)

// Convert pair from consul to map
func convertPairsToObject(prefix string, pairs api.KVPairs) (map[string]any, error) {
	config := make(map[string]any)
	trimPrefix := fmt.Sprintf("%s%s", prefix, pathSeparator)
	for _, pair := range pairs {
		key := strings.TrimPrefix(pair.Key, trimPrefix)
		if len(key) == 0 {
			continue
		}
		addToObject(config, key, string(pair.Value))
	}

	return config, nil
}

func addToObject(config map[string]any, parentKey string, value string) {
	keys := strings.Split(parentKey, pathSeparator)
	childSection := config
	for i, key := range keys {
		if i < len(keys)-1 {
			nextMap, ok := childSection[key]
			if !ok {
				nextMap = make(map[string]any)
				childSection[key] = nextMap
			} else {
				// rewrite node if it's created is least node
				if _, ok := nextMap.(map[string]any); !ok {
					nextMap = make(map[string]any)
					childSection[key] = nextMap
					logger.Warn(context.Background(),
						"consul: duplicate key and path, key was overridden",
						logger.String("full_key", parentKey),
						logger.String("duplicate_key", key),
					)
				}
			}
			childSection = nextMap.(map[string]any)
		} else {
			if _, ok := childSection[key].(map[string]any); !ok {
				childSection[key] = value
			} else {
				logger.Warn(context.Background(),
					"consul: duplicate key and path, key was skipped",
					logger.String("full_key", parentKey),
					logger.String("duplicate_key", key),
				)
			}
		}
	}
}

// Convert from consul to map
func convertObjectToKeys(prefix string, root map[string]any) (map[string][]byte, error) {
	if root == nil {
		return nil, nil
	}

	res := make(map[string][]byte)

	var dfs func(parentPath string, config any) error

	dfs = func(parentPath string, config any) error {
		switch cfg := config.(type) {
		case map[string]any:
			for key, cfg := range cfg {
				nextKey := fmt.Sprintf("%s%s%s", parentPath, pathSeparator, key)
				dfs(nextKey, cfg)
			}
		default:
			val, err := anyToBytes(cfg)
			if err != nil {
				return err
			}
			res[parentPath] = val
		}
		return nil
	}

	err := dfs(prefix, root)
	return res, err
}

func anyToBytes(data any) ([]byte, error) {
	if data == nil {
		return []byte{}, nil
	}

	switch v := data.(type) {
	case []byte:
		return v, nil

	case string:
		return []byte(v), nil

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return []byte(fmt.Sprintf("%d", v)), nil

	case float32, float64:
		return []byte(fmt.Sprintf("%f", v)), nil

	case bool:
		return []byte(fmt.Sprintf("%t", v)), nil

	case []string:
		return []byte(strings.Join(v, ",")), nil

	case []int:
		strSlice := make([]string, len(v))
		for i, num := range v {
			strSlice[i] = fmt.Sprintf("%d", num)
		}
		return []byte(strings.Join(strSlice, ",")), nil

	case []int64:
		strSlice := make([]string, len(v))
		for i, num := range v {
			strSlice[i] = fmt.Sprintf("%d", num)
		}
		return []byte(strings.Join(strSlice, ",")), nil

	case []float64:
		strSlice := make([]string, len(v))
		for i, num := range v {
			strSlice[i] = fmt.Sprintf("%f", num)
		}
		return []byte(strings.Join(strSlice, ",")), nil

	case []bool:
		strSlice := make([]string, len(v))
		for i, b := range v {
			strSlice[i] = fmt.Sprintf("%t", b)
		}
		return []byte(strings.Join(strSlice, ",")), nil

	default:
		return nil, fmt.Errorf("type is not supported: %T", v)
	}
}
