package kafka

import (
	"github.com/segmentio/kafka-go"
)

type ConsumeOption func(ReaderConfig) ReaderConfig

type ConsumerOffset = int64

var (
	FirstOffset ConsumerOffset = kafka.FirstOffset // The least recent offset available for a partition.
	LastOffset  ConsumerOffset = kafka.LastOffset  // The most recent offset available for a partition.
)

func WithOffsetMode(offset ConsumerOffset) ConsumeOption {
	return func(conf ReaderConfig) ReaderConfig {
		conf.StartOffset = offset
		return conf
	}
}

func WithMaxAttempts(n int) ConsumeOption {
	return func(conf ReaderConfig) ReaderConfig {
		conf.MaxAttempts = n
		return conf
	}
}
