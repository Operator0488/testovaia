package etcd

import (
	"context"
	"fmt"
	"strings"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
)

const (
	pathSeparator = "/"
)

func convertToObject(prefix string, pairs map[string]string) map[string]any {
	config := make(map[string]any)
	trimPrefix := prefix + pathSeparator
	for key, value := range pairs {
		key = strings.TrimPrefix(key, trimPrefix)
		if len(key) == 0 {
			continue
		}
		addToObject(config, key, value)
	}
	return config
}

func convertObjectToKeys(prefix string, root map[string]any) (map[string]string, error) {
	if root == nil {
		return nil, nil
	}

	res := make(map[string]string)

	var dfs func(parentPath string, config any) error
	dfs = func(parentPath string, config any) error {
		switch cfg := config.(type) {
		case map[string]any:
			for key, val := range cfg {
				nextKey := parentPath + pathSeparator + key
				if err := dfs(nextKey, val); err != nil {
					return err
				}
			}
		default:
			res[parentPath] = fmt.Sprintf("%v", cfg)
		}
		return nil
	}

	err := dfs(prefix, root)
	return res, err
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
				if _, ok := nextMap.(map[string]any); !ok {
					nextMap = make(map[string]any)
					childSection[key] = nextMap
					logger.Warn(context.Background(),
						"etcd: duplicate key and path, key was overridden",
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
					"etcd: duplicate key and path, key was skipped",
					logger.String("full_key", parentKey),
					logger.String("duplicate_key", key),
				)
			}
		}
	}
}
