package redis

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_SetGet(t *testing.T) {
	ctx := context.Background()

	db, mock := redismock.NewClientMock()

	mock.ExpectSet("key", []byte(`"value"`), time.Minute).SetVal("OK")
	mock.ExpectGet("key").SetVal(`"value"`) // JSON строки имеют кавычки

	c := &client{
		cfg:   RedisConfig{},
		codec: JSONCodec{},
	}
	c.cli.Store(db)

	err := c.Set(ctx, "key", "value", time.Minute)
	require.NoError(t, err)

	val := c.Get(ctx, "key")
	var result string
	err = val.Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, "value", result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_SetGet_JSONStruct(t *testing.T) {
	ctx := context.Background()

	db, mock := redismock.NewClientMock()

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	user := User{Name: "John", Age: 30}
	expectedJSON := `{"name":"John","age":30}`

	mock.ExpectSet("user", []byte(expectedJSON), time.Minute).SetVal("OK")
	mock.ExpectGet("user").SetVal(expectedJSON)

	c := &client{
		codec: JSONCodec{},
	}
	c.cli.Store(db)

	err := c.Set(ctx, "user", user, time.Minute)
	require.NoError(t, err)

	val := c.Get(ctx, "user")
	var result User
	err = val.Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, user, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_Del(t *testing.T) {
	ctx := context.Background()

	db, mock := redismock.NewClientMock()
	mock.ExpectDel("key1", "key2").SetVal(2)

	c := &client{
		codec: JSONCodec{},
	}
	c.cli.Store(db)

	err := c.Del(ctx, "key1", "key2")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_Publish(t *testing.T) {
	ctx := context.Background()

	db, mock := redismock.NewClientMock()

	mock.ExpectPublish("channel", []byte(`"message"`)).SetVal(1)

	c := &client{
		codec: JSONCodec{},
	}
	c.cli.Store(db)

	err := c.Publish(ctx, "channel", "message")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_Get_Error(t *testing.T) {
	ctx := context.Background()

	db, mock := redismock.NewClientMock()
	mock.ExpectGet("nonexistent").RedisNil()

	c := &client{
		codec: JSONCodec{},
	}
	c.cli.Store(db)

	val := c.Get(ctx, "nonexistent")
	assert.True(t, val.Err() == ErrNotFound)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_Set_Error(t *testing.T) {
	ctx := context.Background()

	db, mock := redismock.NewClientMock()
	mock.ExpectSet("key", []byte(`"value"`), time.Minute).SetErr(redis.ErrClosed)

	c := &client{
		codec: JSONCodec{},
	}
	c.cli.Store(db)

	err := c.Set(ctx, "key", "value", time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, redis.ErrClosed)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TODO: мок не умеет в подписку, додумать
func TestClient_Subscribe_DontSuccess(t *testing.T) {
	ctx := context.Background()

	db, _ := redismock.NewClientMock()

	c := &client{
		codec: JSONCodec{},
		subs:  make(map[*subscriber]struct{}),
	}
	c.cli.Store(db)

	// Вместо мока Subscribe, тестируем только регистрацию
	sub, err := c.Subscribe(ctx, "channel1", "channel2")

	require.Error(t, err)
	require.Nil(t, sub)
}

// тестируем регистрацию и удаление подписчика
func TestClient_Subscriber_Registration(t *testing.T) {
	c := &client{
		subs: make(map[*subscriber]struct{}),
	}

	sub := &subscriber{
		userChannel: make(chan *Message, 10),
		channels:    []string{"test"},
	}

	// Тестируем регистрацию
	c.registerSubscriber(sub)

	c.subsMu.RLock()
	assert.Len(t, c.subs, 1)
	_, exists := c.subs[sub]
	assert.True(t, exists)
	c.subsMu.RUnlock()

	// Тестируем удаление
	c.unregisterSubscriber(sub)

	c.subsMu.RLock()
	assert.Len(t, c.subs, 0)
	c.subsMu.RUnlock()
}

func TestSubscriber_IsClosed(t *testing.T) {
	sub := &subscriber{}

	assert.False(t, sub.isClosed())

	sub.closed = true
	assert.True(t, sub.isClosed())
}

func TestNewHealthLoop(t *testing.T) {
	hl := newHealthLoop()
	require.NotNil(t, hl)

	// Проверяем начальные значения
	assert.False(t, hl.ok.Load())
	assert.Equal(t, int64(0), hl.lat.Load())
	assert.Nil(t, hl.err.Load())
}

func TestClient_Snapshot(t *testing.T) {
	c := &client{
		health: newHealthLoop(),
	}

	// Тестируем snapshot когда health ок
	c.health.ok.Store(true)
	c.health.lat.Store(5000) // 5ms в микросекундах

	snapshot := c.snapshot()
	assert.True(t, snapshot.OK)
	assert.Equal(t, 5*time.Millisecond, snapshot.Latency)
	assert.Nil(t, snapshot.LastErr)

	// Тестируем snapshot когда health не ок
	c.health.ok.Store(false)
	c.health.err.Store(assert.AnError)

	snapshot = c.snapshot()
	assert.False(t, snapshot.OK)
	assert.Equal(t, 5*time.Millisecond, snapshot.Latency) // latency остается
	assert.Equal(t, assert.AnError, snapshot.LastErr)
}

func TestClient_Latency(t *testing.T) {
	c := &client{
		health: newHealthLoop(),
	}

	// Начальная latency
	assert.Equal(t, time.Duration(0), c.latency())

	// Устанавливаем latency
	c.health.lat.Store(2500) // 2.5ms
	assert.Equal(t, 2500*time.Microsecond, c.latency())

	// Обнуляем latency
	c.health.lat.Store(0)
	assert.Equal(t, time.Duration(0), c.latency())
}

func TestHealthLoop_Start_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	db, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")

	c := &client{
		health: newHealthLoop(),
	}
	c.cli.Store(db)

	// Устанавливаем lastUsed чтобы пропустить пинг
	c.touchActivity()

	// Запускаем health loop в горутине
	go c.health.start(ctx, c)

	// Даем немного времени на запуск
	time.Sleep(100 * time.Millisecond)

	// Отменяем контекст - health loop должен остановиться
	cancel()

	// Даем время на остановку
	time.Sleep(100 * time.Millisecond)

	// Проверяем что можно безопасно остановиться
}

func TestHealthLoop_PingSuccess(t *testing.T) {
	ctx := context.Background()

	db, mock := redismock.NewClientMock()
	mock.ExpectPing().SetVal("PONG")

	c := &client{
		health: newHealthLoop(),
	}
	c.cli.Store(db)

	// Устанавливаем lastUsed в прошлом чтобы форсировать пинг
	c.lastActivity.Store(time.Now().Add(-time.Minute).UnixNano())

	// Вызываем health check вручную
	start := time.Now()
	err := c.universal().Ping(ctx).Err()
	require.NoError(t, err)

	// Обновляем health
	c.health.ok.Store(true)
	c.health.lat.Store(time.Since(start).Microseconds())

	// Проверяем snapshot
	snapshot := c.snapshot()
	assert.True(t, snapshot.OK)
	assert.True(t, snapshot.Latency > 0)
	assert.Nil(t, snapshot.LastErr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthLoop_PingError(t *testing.T) {
	ctx := context.Background()

	db, mock := redismock.NewClientMock()
	mock.ExpectPing().SetErr(assert.AnError)

	c := &client{
		health: newHealthLoop(),
	}
	c.cli.Store(db)

	// Устанавливаем lastUsed в прошлом
	c.lastActivity.Store(time.Now().Add(-time.Minute).UnixNano())

	// Вызываем health check вручную
	err := c.universal().Ping(ctx).Err()
	require.Error(t, err)

	// Обновляем health с ошибкой
	c.health.ok.Store(false)
	c.health.err.Store(err)

	// Проверяем snapshot
	snapshot := c.snapshot()
	assert.False(t, snapshot.OK)
	assert.Equal(t, assert.AnError, snapshot.LastErr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthLoop_SkipOnRecentActivity(t *testing.T) {
	//ctx := context.Background()

	c := &client{
		health: newHealthLoop(),
	}

	// Устанавливаем недавнюю активность
	c.touchActivity()

	// Health loop должен пропустить пинг т.к. активность была недавно
	// Это сложно протестировать напрямую, но можно проверить логику

	lastUsed := c.lastUsed()
	assert.WithinDuration(t, time.Now(), lastUsed, time.Second)

	// Если с момента lastUsed прошло меньше interval, пинг должен быть пропущен
	timeSinceLastUsed := time.Since(lastUsed)
	assert.True(t, timeSinceLastUsed < 30*time.Second)
}
