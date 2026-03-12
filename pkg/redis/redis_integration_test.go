//go:build integration

package redis

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ДЛЯ ЗАПУСКА ВКЛЮЧАЕМ ДОКЕР
func TestRedisIntegration_WithDocker(t *testing.T) {

	log.Println("=== START TESTS ===")

	ctx := context.Background()

	confOne, stopOne, err := startTestRedisVar1(ctx)
	if err != nil {
		t.Fatal("err start redis:", err)
	}
	defer stopOne()

	confTwo, stopTwo, err := startTestRedisVar2(ctx)
	if err != nil {
		t.Fatal("err start redis:", err)
	}
	defer stopTwo()

	cl, err := New(ctx, confOne)
	if err != nil {
		t.Fatal("err new client:", err)
	}

	errS := basicOperations(ctx, cl)

	if len(errS) > 0 {
		for _, e := range errS {
			t.Error(e)
		}
		t.Fatal("basic operations failed")
	}

	errS = pubsubOperations(ctx, cl)

	if len(errS) > 0 {
		for _, e := range errS {
			t.Error(e)
		}
		t.Fatal("pubsub operations failed")
	}

	log.Println("=== START TESTS WITH REBUILD ===")

	err = cl.(*client).rebildClient(ctx, confTwo)
	if err != nil {
		t.Fatal("err rebuild client:", err)
	}

	errS = basicOperations(ctx, cl)

	if len(errS) > 0 {
		for _, e := range errS {
			t.Error(e)
		}
		t.Fatal("basic operations after rebuild failed")
	}

	errS = pubsubOperationsWithRebuild(ctx, cl, confOne)

	if len(errS) > 0 {
		for _, e := range errS {
			t.Error(e)
		}
		t.Fatal("basic operations with rebuild failed")
	}

}

func pubsubOperations(ctx context.Context, client Redis) []error {

	log.Println("=== Pubsub Operations START ===")
	var errSlice []error

	sub, err := client.Subscribe(ctx, "info", "order", "done")

	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Sub Operations Error: %v", err))
	}

	var user = TestUser{
		12345,
		"Igor",
	}

	err = client.Publish(ctx, "order", user)
	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Pub Operations Error: %v", err))
	}

	ch := sub.Channel()

	select {
	case msg := <-ch:
		log.Println(msg.Channel, msg.Payload)
	}

	err = sub.Close()
	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Close Operations Error: %v", err))
	}

	log.Println("=== Pubsub Operations END ===")

	return errSlice

}

func pubsubOperationsWithRebuild(ctx context.Context, cl Redis, conf RedisConfig) []error {

	log.Println("=== Pubsub Operations With Rebuild START ===")
	var errSlice []error

	sub, err := cl.Subscribe(ctx, "info", "order", "done")

	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Sub Operations Error: %v", err))
	}

	var user = TestUser{
		123,
		"Ifor",
	}

	err = cl.Publish(ctx, "order", user)
	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Pub Operations Error: %v", err))
	}

	ch := sub.Channel()

	go func() {
		timer := time.After(10 * time.Second)
		for {
			select {
			case msg := <-ch:
				log.Println(msg.Channel, msg.Payload)
			case <-timer:
				errSlice = append(errSlice, fmt.Errorf("no messages received after rebuild"))
				return
			}
		}
	}()

	time.Sleep(2 * time.Second) // ждем немного, чтобы прочитать из канала, на всякий

	err = cl.(*client).rebildClient(ctx, conf)
	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("rebuild cl error: %v", err))
	}

	err = cl.Publish(ctx, "info", user)
	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Pub Operations Error: %v", err))
	}

	time.Sleep(2 * time.Second) // ждем немного, чтобы прочитать из канала, на всякий

	err = sub.Close()
	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Close Operations Error: %v", err))
	}

	log.Println("=== Pubsub Operations With Rebuild END ===")

	return errSlice

}

func basicOperations(ctx context.Context, client Redis) []error {

	log.Println("=== Basic Operations START ===")

	var errSlice []error
	var got = TestUser{
		123,
		"Ifor",
	}

	log.Println("=== Set Operations START ===")

	err := client.Set(ctx, "key1", got, 10*time.Minute)
	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Set Operations Error: %v", err))
	}
	log.Println("=== Set Operations END ===")

	log.Println("=== Get Operations START ===")
	result := client.Get(ctx, "key1")

	var expected TestUser

	if result.Err() != nil {

		errSlice = append(errSlice, fmt.Errorf("Get Operations Error: %v", result.Err()))

	} else {

		err = result.Scan(&expected)

		if err != nil {
			log.Printf("error get: %v", err)
			errSlice = append(errSlice, fmt.Errorf("Get Operations Error/ Scan: %v", err))

		}

	}

	if got != expected {
		errSlice = append(errSlice, fmt.Errorf("Get Operations Mismatch: expected %+v, got %+v", expected, got))
	}

	log.Println("=== Get Operations END ===")

	log.Println("=== DEL Operations START ===")
	err = client.Del(ctx, "key1")
	if err != nil {
		errSlice = append(errSlice, fmt.Errorf("Del Operations error: %v", err))

	}

	result = client.Get(ctx, "key1")
	if result.Err() == nil {
		errSlice = append(errSlice, fmt.Errorf("Del Operations get error"))

	}
	log.Println("=== DEL Operations END ===")

	log.Println("=== Basic Operations END ===")

	return errSlice
}

func startTestRedisVar1(ctx context.Context) (RedisConfig, func(), error) {

	log.Println("=== START: startTestRedisVar1 ===")

	req := testcontainers.ContainerRequest{
		Image:        "redis:7",
		ExposedPorts: []string{"6379/tcp"},
		Env: map[string]string{
			"REDIS_PASSWORD": "supersecretpassword",
		},
		Cmd: []string{
			"redis-server",
			"--requirepass", "supersecretpassword",
		},
		WaitingFor: wait.ForListeningPort("6379/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return RedisConfig{}, nil, err
	}

	// получаем адрес
	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")
	addr := fmt.Sprintf("%s:%s", host, port.Port())

	cfg := RedisConfig{
		Addrs:        []string{addr},
		DB:           0,
		PoolSize:     20,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		Username:     "",
		Password:     "supersecretpassword",
	}

	// функция для остановки
	cleanup := func() {
		_ = container.Terminate(ctx)
	}

	log.Println("=== END: startTestRedisVar1 ===")

	return cfg, cleanup, nil
}

func startTestRedisVar2(ctx context.Context) (RedisConfig, func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7",
		ExposedPorts: []string{"6379/tcp"},
		Env: map[string]string{
			"REDIS_PASSWORD": "supersecretpasswordTWO",
		},
		Cmd: []string{
			"redis-server",
			"--requirepass", "supersecretpasswordTWO",
		},
		WaitingFor: wait.ForListeningPort("6379/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return RedisConfig{}, nil, err
	}

	// получаем адрес
	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")
	addr := fmt.Sprintf("%s:%s", host, port.Port())

	cfg := RedisConfig{
		Addrs:        []string{addr},
		DB:           0,
		PoolSize:     20,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		Username:     "",
		Password:     "supersecretpasswordTWO",
	}

	// функция для остановки
	cleanup := func() {
		_ = container.Terminate(ctx)
	}

	log.Println("Test Redis started")

	return cfg, cleanup, nil
}
