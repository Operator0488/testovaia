package trace

import (
	"errors"

	"git.vepay.dev/knoknok/backend-platform/pkg/config"
)

type TracingConfig struct {
	Endpoint    string  // прим: "tempo:4317"/ "localhost:4317"
	Insecure    bool    // true для локальной без TLS
	SampleRatio float64 //трассируем: 1 — все, 0 — ничего
	ServiceName string  //идентификатор имени сервиса
	ServiceEnv  string  // енв
	ServiceVer  string  // версия приложения
	Username    string
	Password    string
	Protocol    string // протокол otlp либо grpc либо http/protobuf
	//UseCollector bool // использовать коллектор или нет
}

func GetTracingConfig(a config.Configurer) (TracingConfig, error) {
	conf := TracingConfig{
		Endpoint:    a.GetString("trace.endpoint"),
		Insecure:    a.GetBool("trace.insecure"),
		SampleRatio: a.GetFloat64("trace.sample_ratio"),
		ServiceName: a.GetStringOrDefault("trace.sevice_name", a.GetString(config.EnvAppName)),
		ServiceEnv:  a.GetString("trace.env"),
		ServiceVer:  a.GetString("trace.service_ver"),
		Protocol:    a.GetString("trace.protocol"),
	}

	if err := conf.Validate(); err != nil {
		return TracingConfig{}, err
	}

	return conf, nil
}

func (s *TracingConfig) Validate() error {
	if len(s.Endpoint) == 0 {
		return errors.New("trace.endpoint is required")
	}

	return nil
}
