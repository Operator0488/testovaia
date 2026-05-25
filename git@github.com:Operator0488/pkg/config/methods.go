package config

import "time"

func GetString(key string) string {
	return configInstance.GetString(key)
}

func GetInt(key string) int {
	return configInstance.GetInt(key)
}

func GetBool(key string) bool {
	return configInstance.GetBool(key)
}

func GetStringSlice(key string) []string {
	return configInstance.GetStringSlice(key)
}

func GetIntSlice(key string) []int {
	return configInstance.GetIntSlice(key)
}

func GetDuration(key string) time.Duration {
	return configInstance.GetDuration(key)
}

func GetFloat64(key string) float64 {
	return configInstance.GetFloat64(key)
}
