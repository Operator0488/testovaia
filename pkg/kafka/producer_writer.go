package kafka

import (
	"errors"
	"net"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	defaultMetaTTL             = 6 * time.Second
	defaultIdleTimeout         = 30 * time.Second
	defaultMaxAttemptsDelivery = 10
	defaultBatchSize           = 100
	defaultBatchTimeout        = 5 * time.Millisecond
	defaultRequireAck          = RequireOne
)

type RequiredAcks int
type CleanupPolicy string

const (
	RequireNone RequiredAcks = 0
	RequireOne  RequiredAcks = 1
	RequireAll  RequiredAcks = -1

	CleanupPolicyCompact  CleanupPolicy = "compact"
	CleanupPolicyDelete   CleanupPolicy = "delete"
	CleanupPolicyCombined CleanupPolicy = "compact,delete"
)

func isValidCleanupPolicy(policy CleanupPolicy) bool {
	if len(policy) == 0 {
		return true
	}
	switch policy {
	case CleanupPolicyCompact, CleanupPolicyDelete, CleanupPolicyCombined:
		return true
	default:
		return false
	}
}

type ProducerConfig struct {
	// Time limit on how often incomplete message batches will be flushed to
	// kafka.
	//
	// The default is to flush at least every second.
	BatchTimeout time.Duration

	// AllowAutoTopicCreation notifies writer to create topic if missing.
	CreatingConfig CreateTopicConfig
}

type CreateTopicConfig struct {
	AllowCreate bool
	// CleanupPolicy must be "compact", "delete" or "compact,delete"
	CleanupPolicy CleanupPolicy
	// ReplicationFactor for the topic. -1 or 0 indicates unset.
	ReplicationFactor int
	// NumPartitions created. -1 or 0 indicates unset.
	NumPartitions int
}

func createWriter(brokers []string, kafkaDialer *dialer, config ProducerConfig) (*kafka.Writer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("brokers not specified")
	}

	dialer := (&net.Dialer{
		Timeout:       kafkaDialer.Timeout,
		Deadline:      kafkaDialer.Deadline,
		LocalAddr:     kafkaDialer.LocalAddr,
		DualStack:     kafkaDialer.DualStack,
		FallbackDelay: kafkaDialer.FallbackDelay,
		KeepAlive:     kafkaDialer.KeepAlive,
	})

	transport := &kafka.Transport{
		Dial:        dialer.DialContext,
		SASL:        kafkaDialer.SASLMechanism,
		TLS:         kafkaDialer.TLS,
		ClientID:    kafkaDialer.ClientID,
		IdleTimeout: defaultIdleTimeout,
		MetadataTTL: defaultMetaTTL,
	}

	batchTimeout := defaultBatchTimeout
	if config.BatchTimeout != 0 {
		batchTimeout = config.BatchTimeout
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		MaxAttempts:  defaultMaxAttemptsDelivery,
		BatchSize:    defaultBatchSize,
		Balancer:     &kafka.RoundRobin{},
		BatchTimeout: batchTimeout,
		RequiredAcks: kafka.RequiredAcks(defaultRequireAck),
		Transport:    transport,
	}

	return w, nil
}
