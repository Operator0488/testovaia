package kafka

import "time"

type ProducerOption func(ProducerConfig) ProducerConfig

func WithTopicCreate(cfg CreateTopicConfig) ProducerOption {
	return func(conf ProducerConfig) ProducerConfig {
		conf.CreatingConfig = cfg
		conf.CreatingConfig.AllowCreate = true
		return conf
	}
}

func WithBatchTimeout(batchTimeout time.Duration) ProducerOption {
	return func(conf ProducerConfig) ProducerConfig {
		conf.BatchTimeout = batchTimeout
		return conf
	}
}
