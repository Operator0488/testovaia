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
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/etcd"
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

func TestProvider_SetAndGet(t *testing.T) {
	ctx := context.Background()
	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := etcd.NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	provider := NewProvider(client, "app")

	input := map[string]any{
		"db": map[string]any{
			"host": "localhost",
			"port": "5432",
		},
		"kafka": map[string]any{
			"topic": "events",
		},
	}

	err = provider.Set(ctx, input)
	require.NoError(t, err)

	result, err := provider.Get(ctx)
	require.NoError(t, err)

	assert.Equal(t, "localhost", getValueByPath(result, "db", "host"))
	assert.Equal(t, "5432", getValueByPath(result, "db", "port"))
	assert.Equal(t, "events", getValueByPath(result, "kafka", "topic"))
}

func TestProvider_Get_WithExistingData(t *testing.T) {
	ctx := context.Background()
	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	// предзаполняем через raw client
	rawClient, err := clientv3.New(cfg)
	require.NoError(t, err)
	_, err = rawClient.Put(ctx, "myapp/db/host", "pghost")
	require.NoError(t, err)
	_, err = rawClient.Put(ctx, "myapp/db/port", "5433")
	require.NoError(t, err)
	rawClient.Close()

	client, err := etcd.NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	provider := NewProvider(client, "myapp")

	result, err := provider.Get(ctx)
	require.NoError(t, err)

	assert.Equal(t, "pghost", getValueByPath(result, "db", "host"))
	assert.Equal(t, "5433", getValueByPath(result, "db", "port"))
}

func TestProvider_Watch_ReceivesChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := etcd.NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	provider := NewProvider(client, "app")

	ch := make(chan map[string]any, 10)
	err = provider.Watch(ctx, func(data map[string]interface{}) {
		ch <- data
	})
	require.NoError(t, err)

	// initial snapshot (пустой)
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// изменяем через raw client
	rawClient, err := clientv3.New(cfg)
	require.NoError(t, err)
	defer rawClient.Close()

	_, err = rawClient.Put(ctx, "app/db/host", "newhost")
	require.NoError(t, err)

	select {
	case data := <-ch:
		assert.Equal(t, "newhost", getValueByPath(data, "db", "host"))
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for watch callback")
	}
}

func TestProvider_Set_Idempotent(t *testing.T) {
	ctx := context.Background()
	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := etcd.NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	provider := NewProvider(client, "app")

	// 1. Первичная запись
	err = provider.Set(ctx, map[string]any{
		"db": map[string]any{
			"host": "localhost",
			"port": "5432",
		},
	})
	require.NoError(t, err)

	rawClient, err := clientv3.New(cfg)
	require.NoError(t, err)
	defer rawClient.Close()

	resp, err := rawClient.Get(ctx, "app/", clientv3.WithPrefix(), clientv3.WithLimit(1))
	require.NoError(t, err)
	revisionAfterFirstSet := resp.Header.Revision

	// 2. Повторный Set с теми же данными — не должен ничего писать
	err = provider.Set(ctx, map[string]any{
		"db": map[string]any{
			"host": "localhost",
			"port": "5432",
		},
	})
	require.NoError(t, err)

	resp, err = rawClient.Get(ctx, "app/", clientv3.WithPrefix(), clientv3.WithLimit(1))
	require.NoError(t, err)
	assert.Equal(t, revisionAfterFirstSet, resp.Header.Revision,
		"revision should not change when setting identical data")

	// 3. Повторный Set: изменённое значение существующего ключа + новый ключ
	err = provider.Set(ctx, map[string]any{
		"db": map[string]any{
			"host": "CHANGED",
			"port": "5432",
			"name": "mydb",
		},
	})
	require.NoError(t, err)

	resp, err = rawClient.Get(ctx, "app/", clientv3.WithPrefix(), clientv3.WithLimit(1))
	require.NoError(t, err)
	revisionAfterSecondSet := resp.Header.Revision

	// 4. Проверяем: revision выросла ровно на 1 (только новый ключ db.name)
	assert.Equal(t, revisionAfterFirstSet+1, revisionAfterSecondSet,
		"revision should increase by exactly 1 (only new key written)")

	// 5. Существующий ключ НЕ перезаписан
	result, err := provider.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "localhost", getValueByPath(result, "db", "host"),
		"existing key must not be overwritten")

	// 6. Новый ключ записан
	assert.Equal(t, "mydb", getValueByPath(result, "db", "name"),
		"new key must be written")
}

func TestProvider_PrefixIsolation(t *testing.T) {
	ctx := context.Background()
	cfg, cleanup, err := startEtcdContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	client, err := etcd.NewClientWithConfig(cfg)
	require.NoError(t, err)
	defer client.Close()

	providerA := NewProvider(client, "serviceA")
	providerB := NewProvider(client, "serviceB")

	err = providerA.Set(ctx, map[string]any{
		"db": map[string]any{"host": "hostA"},
	})
	require.NoError(t, err)

	resultB, err := providerB.Get(ctx)
	require.NoError(t, err)
	assert.Empty(t, resultB)

	resultA, err := providerA.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "hostA", getValueByPath(resultA, "db", "host"))
}

