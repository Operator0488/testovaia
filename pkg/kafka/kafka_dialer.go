package kafka

import (
	"crypto/tls"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
)

type dialer = kafka.Dialer
type Mechanism = sasl.Mechanism

const (
	dialerTimeout  = 30 * time.Second
	dialerClientID = "kafka_client_id"
)

func newDialer(
	TLS *tls.Config,
	SASLMechanism Mechanism,
) dialer {
	return dialer{
		ClientID:      dialerClientID,
		TLS:           TLS,
		SASLMechanism: SASLMechanism,
		Timeout:       dialerTimeout,
	}
}
