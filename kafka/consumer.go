package kafka

import (
	"context"
	"fmt"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"github.com/segmentio/kafka-go"
)

const (
	defaultHeartbeatInterval = 3 * time.Second
	defaultMaxAttempts       = 3
)

type Message = kafka.Message
type ReaderConfig = kafka.ReaderConfig
type ConsumeHandler func(ctx context.Context, msg Message) error
type consumeMiddleware func(ctx context.Context, msg Message, next ConsumeHandler) error

// reader is a minimal interface implemented by *kafka.reader that the consumer depends on.
type reader interface {
	FetchMessage(ctx context.Context) (Message, error)
	CommitMessages(ctx context.Context, msgs ...Message) error
	Close() error
}

type Consumer interface {
	Init(ctx context.Context, dialer *dialer, brokers []string) error
	Run(ctx context.Context) error
	Close() error
}

type consumer struct {
	config      kafka.ReaderConfig
	reader      reader
	handler     ConsumeHandler
	middlewares []consumeMiddleware
}

// newConsumer only create consumer with config.
func newConsumer(
	topic, groupID string,
	handler ConsumeHandler,
	opts ...ConsumeOption,
) (*consumer, error) {
	if handler == nil {
		return nil, fmt.Errorf("handler not defined")
	}

	if len(topic) == 0 {
		return nil, fmt.Errorf("topic not defined")
	}

	config := ReaderConfig{
		GroupID:           groupID,
		Topic:             topic,
		Logger:            &loggerWrap{},
		ErrorLogger:       &loggerWrap{errors: true},
		HeartbeatInterval: defaultHeartbeatInterval,
		MaxAttempts:       defaultMaxAttempts,
		IsolationLevel:    kafka.ReadCommitted,
		StartOffset:       kafka.LastOffset,
	}

	for _, o := range opts {
		config = o(config)
	}

	c := &consumer{
		config:  config,
		handler: handler,
	}

	return c, nil
}

// Start run topic listener.
func (c *consumer) Run(ctx context.Context) error {
	ctx = logger.With(ctx, logger.String("topic", c.config.Topic))
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				logger.Error(ctx, "Error fetching message",
					logger.Err(err),
				)
				continue
			}

			consumeFunc := c.createConsumeChain()

			if err := consumeFunc(ctx, msg); err != nil {
				logger.Error(ctx, "Error processing message",
					logger.Any("message", msg),
					logger.Err(err),
				)
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				logger.Error(ctx, "Error committing message",
					logger.Err(err),
				)
			}
		}
	}
}

func (c *consumer) createConsumeChain() ConsumeHandler {
	baseHandler := c.handler

	handler := baseHandler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		middleware := c.middlewares[i]
		next := handler
		handler = func(ctx context.Context, message Message) error {
			return middleware(ctx, message, next)
		}
	}

	return handler
}

func (c *consumer) Use(middleware consumeMiddleware) {
	c.middlewares = append(c.middlewares, middleware)
}

// Init apply common properties for config.
func (c *consumer) Init(
	ctx context.Context,
	dialer *dialer,
	brokers []string,
) error {

	c.config.Brokers = brokers
	c.config.Dialer = dialer
	c.reader = kafka.NewReader(c.config)
	return nil
}

// Close finish him
func (c *consumer) Close() error {
	if c.reader == nil {
		return nil
	}
	return c.reader.Close()
}
