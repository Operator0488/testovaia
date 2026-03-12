package redis

import "time"

type RedisConfig struct {
	Addrs        []string
	DB           int
	PoolSize     int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Username     string
	Password     string
}
