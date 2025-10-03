package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type produceFunc func(ctx context.Context, messages []Message) error
type produceMiddleware func(ctx context.Context, messages []Message, next produceFunc) error

type Producer interface {
	Produce(ctx context.Context, msg ...ProduceMessage) error
	Close() error
}

type producer struct {
	writer      *kafka.Writer
	topic       string
	middlewares []produceMiddleware
}

// NewProducer create producer by config
// required params:
//   - brokers
//   - topic
//   - dialer
func newProducer(brokers []string, dialer *dialer, topic string, config ProducerConfig) (*producer, error) {
	writer, err := createWriter(brokers, dialer, config)
	if err != nil {
		return nil, err
	}
	writer.Logger = &loggerWrap{}
	writer.ErrorLogger = &loggerWrap{errors: true}
	writer.AllowAutoTopicCreation = false
	return &producer{writer: writer, topic: topic}, nil
}

func (p *producer) Produce(ctx context.Context, messages ...ProduceMessage) error {
	msgs := convertSlice(messages, func(m ProduceMessage) Message {
		msg := Message{
			Key:   m.Key,
			Value: m.Value,
		}
		msg.Topic = p.topic
		return msg
	})

	produceFunc := p.createProduceChain()
	return produceFunc(ctx, msgs)
}

func (p *producer) createProduceChain() produceFunc {
	baseHandler := func(ctx context.Context, messages []Message) error {
		return p.writer.WriteMessages(ctx, messages...)
	}

	handler := baseHandler
	for i := len(p.middlewares) - 1; i >= 0; i-- {
		middleware := p.middlewares[i]
		next := handler
		handler = func(ctx context.Context, messages []Message) error {
			return middleware(ctx, messages, next)
		}
	}

	return handler
}

func (p *producer) Use(middleware produceMiddleware) {
	p.middlewares = append(p.middlewares, middleware)
}

func (p *producer) Close() error {
	return p.writer.Close()
}
