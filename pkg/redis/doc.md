## Документация: Redis-клиент

### Цель
Пакет redis реализует унифицированны компонент для работы с Redis:
- кэширование (Set/Get/Del),
- Pub/Sub,
- health-check с метрикой latency,
- единый интерфейс Redis для интеграции в сервисы.

### Client
Для создания клиента необходимо использовать функцию New:
```go
func New(ctx context.Context, cfg config.RedisConfig) (Redis, error)
```

Пересоздание клиента происходит с помощью функции RebildClient:
- создает новый клиент,
- пингует,
- переподписывает активные подписки,
- закрывает старый клиент,

```go
func (c *client) RebildClient(ctx context.Context, username, password string) error
```

### Основные интерфейсы

#### RedisClient
```go
type Redis interface {
    // базовые команды
    Set(ctx context.Context, key string, val any, ttl time.Duration) error
    Get(ctx context.Context, key string) Value
    Del(ctx context.Context, keys ...string) error

    // pub/sub
    Publish(ctx context.Context, channel string, msg any) error
    Subscribe(ctx context.Context, channels ...string) (Subscriber, error)

    // healthcheck 
	HealthCheck() error

    // завершение работы
    Close() error
}
```

##### Value
Обертка над результатом GET
```go
type Value struct {
    data string
    err  error
}

func (v Value) Scan(dst any) error
func (v Value) Err() error
func (v Value) IsNotFound() bool
}
```
Пример использования:
```go
// SET 
err := client.Set(ctx, "key1", user, 10*time.Minute)
if err != nil {
	log.Printf("Set error: %v", err)
}

// GET
result := client.Get(ctx, "key1")

if result.Err() != nil {
    log.Printf("Get error: %v", result.Err())

} else {

	var val User
	err = result.Scan(&val)
	
	if err != nil {
		log.Printf("error get: %v", err)
	}

	fmt.Println("Got value: ", val.ID, val.Name)
}

// DELETE
err = client.Del(ctx, "key1")
if err != nil {
	log.Printf("error del: %v", err)
}
 ```  


#### Subscriber
```go
type Subscriber interface {
    Channel() <-chan *Message
    Close() error
}
```

##### Message
Обертка над результатом чтения из канала
```go
type Message struct {
    Channel string
    Payload string
}
```

Но при проходе по каналу с помощью range, *возникнет блокировка*, важно учитывать при реализации.
Пример использования:
```go
    sub, err := client.Subscribe(ctx, "info", "order", "done")
	if err != nil {
		log.Printf("Subscribe error: %v", err)
	}

	err = client.Publish(ctx, "order", user)
	if err != nil {
		log.Printf("Publish error: %v", err)
	}

	ch := sub.Channel()

	select {
	case msg := <-ch:
		fmt.Println(msg.Channel, msg.Payload)
	}

	err = sub.Close()
	if err != nil {
		log.Printf("Close error: %v", err)
	}
```

### Healthcheck
Раз в 30 секунд выполняется PING.
Если Redis недавно использовался (< 30s) — пинг пропускается.
При успехе фиксируется Latency.
При ошибке — сохраняется LastErr и OK=false.

```go
type Health struct {
    OK      bool
    Latency time.Duration
    LastErr error
}
```
Метод Snapshot возвращает текущее состояние клиента:
```go
func (c *client) Snapshot() Health
```

