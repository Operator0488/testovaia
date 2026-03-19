package etcd

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
