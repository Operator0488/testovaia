//go:build integration

package etcd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func startEtcdContainer(ctx context.Context) (clientv3.Config, func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        "quay.io/coreos/etcd:v3.5.21",
		ExposedPorts: []string{"2379/tcp"},
		Cmd: []string{
			"etcd",
			"--advertise-client-urls=http://0.0.0.0:2379",
			"--listen-client-urls=http://0.0.0.0:2379",
		},
		WaitingFor: wait.ForListeningPort("2379/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return clientv3.Config{}, nil, err
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "2379")
	endpoint := fmt.Sprintf("%s:%s", host, port.Port())

	cfg := clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 5 * time.Second,
	}

	cleanup := func() {
		_ = container.Terminate(ctx)
	}

	return cfg, cleanup, nil
}

func TestGetByPrefix_Empty(t *testing.T) {
	ctx := context.Background()
	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	result, rev, err := client.GetByPrefix(ctx, "test/")
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Greater(t, rev, int64(0))
}

func TestPutAndGet(t *testing.T) {
	ctx := context.Background()
	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	require.NoError(t, client.Put(ctx, "test/db/host", "localhost"))
	require.NoError(t, client.Put(ctx, "test/db/port", "5432"))
	require.NoError(t, client.Put(ctx, "test/db/name", "mydb"))

	result, _, err := client.GetByPrefix(ctx, "test/")
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "localhost", result["test/db/host"])
	assert.Equal(t, "5432", result["test/db/port"])
	assert.Equal(t, "mydb", result["test/db/name"])
}

func TestGetByPrefix_Isolation(t *testing.T) {
	ctx := context.Background()
	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	require.NoError(t, client.Put(ctx, "app/db/host", "apphost"))
	require.NoError(t, client.Put(ctx, "other/db/host", "otherhost"))

	result, _, err := client.GetByPrefix(ctx, "app/")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "apphost", result["app/db/host"])
}

func TestWatchPrefix_PutEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	ch := make(chan map[string]string, 10)
	err = client.WatchPrefix(ctx, "test/", func(snapshot map[string]string) {
		ch <- snapshot
	})
	require.NoError(t, err)

	// initial snapshot (пустой, т.к. ключей ещё нет)
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// изменяем через отдельный клиент (имитация внешних изменений)
	rawClient, err := clientv3.New(cfg)
	require.NoError(t, err)
	defer rawClient.Close()

	_, err = rawClient.Put(ctx, "test/key1", "value1")
	require.NoError(t, err)

	select {
	case snapshot := <-ch:
		assert.Equal(t, "value1", snapshot["test/key1"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for watch callback")
	}
}

func TestWatchPrefix_DeleteEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	// предзаполняем
	require.NoError(t, client.Put(ctx, "test/key1", "value1"))

	ch := make(chan map[string]string, 10)
	err = client.WatchPrefix(ctx, "test/", func(snapshot map[string]string) {
		ch <- snapshot
	})
	require.NoError(t, err)

	// initial snapshot (содержит key1)
	select {
	case snapshot := <-ch:
		assert.Equal(t, "value1", snapshot["test/key1"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// удаляем через raw client
	rawClient, err := clientv3.New(cfg)
	require.NoError(t, err)
	defer rawClient.Close()

	_, err = rawClient.Delete(ctx, "test/key1")
	require.NoError(t, err)

	select {
	case snapshot := <-ch:
		_, exists := snapshot["test/key1"]
		assert.False(t, exists, "deleted key should not be in snapshot")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for watch callback")
	}
}

func TestWatchPrefix_MultipleUpdates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	ch := make(chan map[string]string, 10)
	err = client.WatchPrefix(ctx, "test/", func(snapshot map[string]string) {
		ch <- snapshot
	})
	require.NoError(t, err)

	// initial snapshot
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// изменяем через отдельный клиент (имитация внешних изменений)
	rawClient, err := clientv3.New(cfg)
	require.NoError(t, err)
	defer rawClient.Close()

	// 4 последовательных Put
	_, err = rawClient.Put(ctx, "test/a", "1")
	require.NoError(t, err)
	_, err = rawClient.Put(ctx, "test/b", "2")
	require.NoError(t, err)
	_, err = rawClient.Put(ctx, "test/c", "3")
	require.NoError(t, err)
	_, err = rawClient.Put(ctx, "test/c", "4")
	require.NoError(t, err)

	// собираем все callbacks, последний должен содержать все ключи
	var lastSnapshot map[string]string
	deadline := time.After(5 * time.Second)
	collected := 0
	for collected < 4 {
		select {
		case snapshot := <-ch:
			lastSnapshot = snapshot
			collected++
		case <-deadline:
			t.Fatalf("timeout: got only %d callbacks, expected 4", collected)
		}
	}

	assert.Equal(t, "1", lastSnapshot["test/a"])
	assert.Equal(t, "2", lastSnapshot["test/b"])
	assert.Equal(t, "4", lastSnapshot["test/c"])
}

func TestWatchPrefix_Compaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	rawClient, err := clientv3.New(cfg)
	require.NoError(t, err)
	defer rawClient.Close()

	// набиваем начальные данные
	for i := 0; i < 5; i++ {
		_, err = rawClient.Put(ctx, fmt.Sprintf("test/key%d", i), fmt.Sprintf("val%d", i))
		require.NoError(t, err)
	}

	// запускаем watcher
	client, err := NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	ch := make(chan map[string]string, 20)
	err = client.WatchPrefix(ctx, "test/", func(snapshot map[string]string) {
		ch <- snapshot
	})
	require.NoError(t, err)

	// initial snapshot
	select {
	case snapshot := <-ch:
		assert.Len(t, snapshot, 5)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// набиваем ещё ревизий
	for i := 5; i < 15; i++ {
		_, err = rawClient.Put(ctx, fmt.Sprintf("test/key%d", i), fmt.Sprintf("val%d", i))
		require.NoError(t, err)
	}

	// забираем все callback'и от Put'ов
	for i := 0; i < 10; i++ {
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for put callback %d", i)
		}
	}

	// запоминаем текущую ревизию и компактим до неё
	resp, err := rawClient.Get(ctx, "test/", clientv3.WithPrefix())
	require.NoError(t, err)
	t.Logf("compacting up to revision %d", resp.Header.Revision)
	_, err = rawClient.Compact(ctx, resp.Header.Revision)
	require.NoError(t, err)

	// теперь добавляем ещё ключ — watcher должен его увидеть
	// (если compaction не сломала watch)
	_, err = rawClient.Put(ctx, "test/after_compact", "works")
	require.NoError(t, err)

	select {
	case snapshot := <-ch:
		t.Logf("got snapshot with %d keys after compaction", len(snapshot))
		assert.Equal(t, "works", snapshot["test/after_compact"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: watcher did not produce snapshot after compaction")
	}
}

func TestWatchPrefix_GracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	callbackCount := 0
	ch := make(chan struct{}, 10)
	err = client.WatchPrefix(ctx, "test/", func(snapshot map[string]string) {
		callbackCount++
		ch <- struct{}{}
	})
	require.NoError(t, err)

	// initial snapshot
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// отменяем контекст — watch должен завершиться без retry
	cancel()
	time.Sleep(500 * time.Millisecond)

	countAfterCancel := callbackCount

	// ждём ещё немного — убеждаемся что callback'ов больше нет (нет retry)
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, countAfterCancel, callbackCount, "no more callbacks after context cancellation")
}
