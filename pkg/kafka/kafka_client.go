package kafka

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	ErrBrokersNotFound       = errors.New("brokers not found")
	ErrProducerNotRegistered = errors.New("producer not registered")
	ErrInvalidCleanupPolicy  = errors.New("invalid cleanup policy")
)

const (
	defaultHealthCheckDuration = time.Second * 30
)

type kafkaClient struct {
	dialer        *dialer
	brokers       []string
	consumers     []Consumer
	producers     map[string]Producer
	consumerGroup string
	mu            *sync.RWMutex
	health        HealthCheker
}

type KafkaClient interface {
	Run(context.Context) error
	HealthCheck(context.Context) error
	RegisterConsumer(
		ctx context.Context,
		topic string,
		handler ConsumeHandler,
		opts ...ConsumeOption,
	) error
	RegisterProducer(
		ctx context.Context,
		topic string,
		opts ...ProducerOption,
	) (Producer, error)
	GetProducer(topic string) (Producer, bool)
	Close() error
}

// NewKafkaClient create kafka client.
func NewKafkaClient(
	brokers []string,
	consumerGroup string,
	opts ...kafkaClientOption,
) (KafkaClient, error) {
	if len(brokers) == 0 {
		return nil, ErrBrokersNotFound
	}

	dialer := newDialer(nil, nil)
	client := &kafkaClient{
		dialer:        &dialer,
		brokers:       brokers,
		consumerGroup: consumerGroup,
		mu:            &sync.RWMutex{},
		producers:     make(map[string]Producer),
	}

	client.health = newHealthChecker(defaultHealthCheckDuration, client.forceHealthCheck)

	for _, o := range opts {
		o(client)
	}

	return client, nil
}

// Run invoke Run method for sub-services.
func (k *kafkaClient) Run(ctx context.Context) error {
	for _, consumer := range k.consumers {
		go consumer.Run(ctx)
	}

	go k.health.Run(ctx)

	return nil
}

// forceHealthCheck tries to establish a connection to any configured broker
// and fetch controller metadata. Returns nil if at least one broker is reachable.
func (k *kafkaClient) forceHealthCheck(ctx context.Context) error {
	var err error
	for _, broker := range k.brokers {
		conn, e := k.dialer.DialContext(ctx, "tcp", broker)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}
		// Ensure connection is closed
		func() {
			defer conn.Close()
			if _, e2 := conn.Controller(); e2 != nil {
				err = errors.Join(err, e2)
			} else {
				err = nil
			}
		}()
		if err == nil {
			return nil
		}
	}
	if err == nil {
		return ErrBrokersNotFound
	}
	return err
}

// HealthCheck return error if kafka not available.
func (k *kafkaClient) HealthCheck(ctx context.Context) error {
	return k.health.GetState().err
}

// RegisterConsumer add listener with callback handler for topic.
func (k *kafkaClient) RegisterConsumer(
	ctx context.Context,
	topic string,
	handler ConsumeHandler,
	opts ...ConsumeOption,
) error {
	consumer, err := newConsumer(topic, k.consumerGroup, handler, opts...)
	if err != nil {
		return err
	}

	if err = consumer.Init(ctx, k.dialer, k.brokers); err != nil {
		return err
	}

	consumer.Use(traceConsumeMiddleware(topic))
	consumer.Use(healthCheckConsumeMiddleware(k.health))

	// Initialize metrics
	metrics.KafkaMessagesTotal.WithLabelValues(topic, "consume").Add(0)
	metrics.KafkaErrorsTotal.WithLabelValues(topic, "consume").Add(0)
	metrics.KafkaLatencySeconds.WithLabelValues(topic, "consume").Observe(0)

	metrics.KafkaConsumerLag.
		WithLabelValues(topic).
		Set(0)

	consumer.Use(metricsConsumeMiddleware())
	k.consumers = append(k.consumers, consumer)
	return nil
}

// RegisterProducer create producer for topic.
func (k *kafkaClient) RegisterProducer(
	ctx context.Context,
	topic string,
	opts ...ProducerOption,
) (Producer, error) {
	if len(topic) == 0 {
		return nil, errors.New("topic not specified")
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	if producer, ok := k.producers[topic]; ok {
		return producer, nil
	}

	config := ProducerConfig{}
	for _, o := range opts {
		config = o(config)
	}

	if config.CreatingConfig.AllowCreate {
		if err := k.createTopic(ctx, topic, config.CreatingConfig); err != nil {
			return nil, err
		}
	}

	producer, err := newProducer(k.brokers, k.dialer, topic, config)
	producer.Use(traceProduceMiddleware(topic))
	producer.Use(healthCheckProducerMiddleware(k.health))

	// Initialize metrics
	metrics.KafkaMessagesTotal.WithLabelValues(topic, "produce").Add(0)
	metrics.KafkaErrorsTotal.WithLabelValues(topic, "produce").Add(0)
	metrics.KafkaLatencySeconds.WithLabelValues(topic, "produce").Observe(0)

	producer.Use(metricsProduceMiddleware())

	if err != nil {
		return nil, err
	}

	k.producers[topic] = producer
	return producer, nil
}

// GetProducer return registered producer for topic.
func (k *kafkaClient) GetProducer(topic string) (Producer, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	producer, ok := k.producers[topic]
	if !ok {
		return nil, false
	}

	return producer, true
}

// createTopic creating topic by config
func (k *kafkaClient) createTopic(ctx context.Context, topic string, config CreateTopicConfig) error {
	dialer := k.dialer
	var conn *kafka.Conn
	var err error

	for _, broker := range k.brokers {
		c, e := dialer.DialContext(ctx, "tcp", broker)
		if e == nil {
			conn = c
			break
		}
		err = errors.Join(err, e)
	}
	if conn == nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))

	ctrlConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer ctrlConn.Close()

	replicationFactor := config.ReplicationFactor
	if replicationFactor == 0 {
		replicationFactor = -1
	}

	numPartitions := config.NumPartitions
	if numPartitions == 0 {
		numPartitions = -1
	}

	cfg := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	}

	if !isValidCleanupPolicy(config.CleanupPolicy) {
		return fmt.Errorf("%w: %s", ErrInvalidCleanupPolicy, config.CleanupPolicy)
	}

	if len(config.CleanupPolicy) > 0 {
		cfg.ConfigEntries = append(cfg.ConfigEntries, kafka.ConfigEntry{
			ConfigName:  "cleanup.policy",
			ConfigValue: string(config.CleanupPolicy),
		})
	}

	return ctrlConn.CreateTopics(cfg)
}

// Close implements KafkaClient.
func (k *kafkaClient) Close() error {
	var err error
	for _, consumer := range k.consumers {
		errC := consumer.Close()
		err = errors.Join(err, errC)
	}

	for _, producer := range k.producers {
		errP := producer.Close()
		err = errors.Join(err, errP)
	}

	return err
}

type kafkaClientOption func(*kafkaClient)

func WithTLS(TLS *tls.Config) kafkaClientOption {
	return func(client *kafkaClient) {
		client.dialer.TLS = TLS
	}
}

func WithSASL(SASLMechanism Mechanism) kafkaClientOption {
	return func(client *kafkaClient) {
		client.dialer.SASLMechanism = SASLMechanism
	}
}
