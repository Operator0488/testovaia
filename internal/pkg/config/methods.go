package config

import (
	"time"

	"github.com/spf13/viper"
)

func (c *Config) GetString(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetString(key)
}

func (c *Config) GetStringOrDefault(key string, def string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.getViperInstance(key).GetString(key)
	if len(v) > 0 {
		return v
	}
	return def
}

func (c *Config) GetBool(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetBool(key)
}

func (c *Config) GetInt(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetInt(key)
}

func (c *Config) GetIntOrDefault(key string, def int) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	inst := c.getViperInstance(key)
	if inst.IsSet(key) {
		return inst.GetInt(key)
	}
	return def
}

func (c *Config) GetBoolOrDefault(key string, def bool) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	inst := c.getViperInstance(key)
	if inst.IsSet(key) {
		return inst.GetBool(key)
	}
	return def
}

func (c *Config) GetInt32(key string) int32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetInt32(key)
}
func (c *Config) GetInt64(key string) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetInt64(key)
}

func (c *Config) GetStringSlice(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetStringSlice(key)
}

func (c *Config) GetIntSlice(key string) []int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetIntSlice(key)
}

func (c *Config) GetDuration(key string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetDuration(key)
}

func (c *Config) GetFloat64(key string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getViperInstance(key).GetFloat64(key)
}

func (c *Config) getViperInstance(key string) *viper.Viper {
	for _, v := range c.storages {
		if v.viper.IsSet(key) {
			return v.viper
		}
	}
	return c.envViper
}
