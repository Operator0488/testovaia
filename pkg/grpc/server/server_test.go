package server

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// подбираем свободный порт
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func testServerConfig(addr string) Config {
	return Config{
		Addr:              addr,
		MaxRecvMsgSize:    4 << 20,
		MaxSendMsgSize:    4 << 20,
		ConnectionTimeout: time.Second,
		KeepAliveTime:     time.Second,
		KeepAliveTimeout:  time.Second,
	}
}

func TestServer_HealthCheck_And_GlobalHealth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr := freePort(t)
	m := New(testServerConfig(addr))

	// до Initialize — ошибка
	if err := m.HealthCheck(ctx); err == nil {
		t.Fatalf("want error before Initialize")
	}

	// Initialize проходит, но до Start  ошибка
	if err := m.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := m.HealthCheck(ctx); err == nil {
		t.Fatalf("want error before Start")
	}

	// Start ок
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck after Start: %v", err)
	}

	// глобальный health в статусе SERVING
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	hc := grpc_health_v1.NewHealthClient(conn)
	resp, err := hc.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check rpc: %v", err)
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("want SERVING, got %v", resp.GetStatus())
	}

	// Остановимся — достаточно, что не паникнуло
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestServer_AddService_IsCalled_OnInitialize(t *testing.T) {
	ctx := context.Background()
	addr := freePort(t)
	m := New(testServerConfig(addr))

	called := false
	m.AddService(func(_ grpc.ServiceRegistrar) { called = true })

	if err := m.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if !called {
		t.Fatalf("service registration func was not called")
	}
	if got, want := m.ServicesCount(), 1; got != want {
		t.Fatalf("ServicesCount: got %d, want %d", got, want)
	}
}

func TestServer_Start_Fails(t *testing.T) {
	ctx := context.Background()
	addr := freePort(t)

	// занимаем порт
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("prelisten: %v", err)
	}
	defer l.Close()

	m := New(testServerConfig(addr))
	if err := m.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := m.Start(ctx); err == nil {
		t.Fatalf("want error on Start when port is busy")
	}
}
