## Компонент для чтения и продюсинга данных в Kafka

### Общее
Пакет предоставляет простой слой над `github.com/segmentio/kafka-go` для регистрации консумеров и продюсеров через единый клиент.

Поддерживается опциональное создание топиков при регистрации продюсера.

### Создание клиента
```go
import (
    "context"
    "log"
    "git.vepay.dev/knoknok/backend-platform/pkg/kafka"
)

func main() {
    brokers := []string{"kafka-1:9092", "kafka-2:9092"}
    groupID := "example-for-tempo-prom-group"

    client, err := kafka.NewKafkaClient(brokers, groupID)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // запуск консумеров
    ctx := context.Background()
    _ = client.Run(ctx)
}
```

### Регистрация консумера
```go
handler := func(ctx context.Context, msg kafka.Message) error {
    // обработка сообщения
    return nil
}

if err := client.RegisterConsumer(context.Background(), "example-for-tempo-prom-topic", handler); err != nil {
    log.Fatal(err)
}
```

Опциональные настройки консумера передаются через `ConsumeOption`.

### Регистрация продюсера и отправка сообщений
```go
// опционально включаем авто‑создание топика при регистрации продюсера
producer, err := client.RegisterProducer(
    context.Background(),
    "example-for-tempo-prom-topic",
    kafka.WithTopicCreate(kafka.CreateTopicConfig{
        CleanupPolicy:     kafka.CleanupPolicyCombined, // "compact,delete"
        ReplicationFactor: 0, // 0 или -1 = взять кластерный default
        NumPartitions:     0, // 0 или -1 = взять кластерный default
    }),
)
if err != nil {
    log.Fatal(err)
}

// отправка сообщений (топик зашит в продюсер при регистрации)
err = producer.Produce(context.Background(),
    kafka.ProduceMessage{Key: []byte("k1"), Value: []byte("v1")},
    kafka.ProduceMessage{Key: []byte("k2"), Value: []byte("v2")},
)
if err != nil { 
    log.Fatal(err)
}
```

### Закрытие клиента
```go
// корректно закрывает все зарегистрированные продюсеры и консумеры
if err := client.Close(); err != nil {
    log.Println("close error:", err)
}
```

### Дополнительно
- Создание топика идемпотентно: если топик существует, то ошибку не возвращаем.
- Если `ReplicationFactor`/`NumPartitions` не заданы (0 или -1), брокер применит кластерные значения по умолчанию (`default.replication.factor`, `num.partitions`).
