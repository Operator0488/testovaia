# Интеграция распределённой трассировки

Интегрирование распределенной трассировка с использованием OpenTelemetry и Grafana Tempo.

## Инициализация трассировки

Конфигурация трассировки загружается с помощью LoadTracingConfig():


````go
func LoadTracingConfig() TracingConfig

type TracingConfig struct {
    Endpoint    string  // прим: "tempo:4317"/ "localhost:4317"
    Insecure    bool    // true для локальной без TLS
    SampleRatio float64 //трассируем: 1 — все, 0 — ничего
    ServiceName string  //идентификатор имени сервиса
    ServiceEnv  string  // енв
    ServiceVer  string  // версия приложения
    Protocol    string // протокол otlp либо grpc либо http/protobuf
}
````

Пакет pkg/tracing содержит функцию InitProvider(cfg).
Настраивает OTLP Exporter, Sampler, Resource (имя сервиса, версия, окружение).
Глобально регистрирует Propagator (TraceContext + Baggage).

Использование в приложении:
````go
 shutdown, err := tracing.InitProvider(ctx, cfg.Tracing)
defer shutdown(ctx)
````
## Kafka

Трассировка сообщений Kafka с помощью otel-kafka-go. Трассировка включается автоматически при создании консьюмера и продюсера:

````go
    groupID := "demo-group"
	kafkaClient, err := kafka.NewKafkaClient(brokers, groupID)  
	if err != nil {
		panic(err)
	}
	defer kafkaClient.Close()
	
	producer, err := kafkaClient.RegisterProducer(ctx,"demo-topic",
	)
	if err != nil {
		panic(err)
	}
````

## HTTP
Трассировка HTTP запросов с помощью otelecho. Трассировка включается c MiddlewareEcho:

````go
    e := echo.New()
	e.Use(tracing.MiddlewareEcho("demo-http"))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request().WithContext(logger.With(c.Request().Context()))
			c.SetRequest(req)
			return next(c)
		}
	})
````

## gRPC
Трассировка gRPC запросов с помощью otelgrpc.

````go
    server := grpc.NewServer(
		grpc.StatsHandler(tracing.ServerHandlerGRPC()), 
		...
    )
````

`````go
    client := grpc.NewClient(
		grpc.StatsHandler(tracing.ClientHandlerGRPC()), 
		...
    )
`````

## Zeebee

Жду 

## BaseApp

Жду 



