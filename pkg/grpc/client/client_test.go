package client

import (
	"context"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/di"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"testing"
	"time"
)

func startHealthServer(t *testing.T) (addr string, stop func()) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = l.Addr().String()

	s := grpc.NewServer()
	hs := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, hs)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go s.Serve(l)

	stop = func() {
		s.Stop()
		_ = l.Close()
	}
	return addr, stop
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	c := di.New()
	ctx := di.WithContainer(context.Background(), c)
	return ctx
}

func TestClient_Manager_Initialize_And_Use(t *testing.T) {
	ctx, cancel := context.WithTimeout(testCtx(t), 5*time.Second)
	defer cancel()

	addr, stop := startHealthServer(t)
	defer stop()

	m := NewManager()

	AddGenericRegistration[grpc_health_v1.HealthClient](m, "health", grpc_health_v1.NewHealthClient)

	resolve := func(service string) Config {
		return Config{
			Address:          addr,
			Timeout:          3 * time.Second,
			MaxRecvMsgSize:   4 << 20,
			MaxSendMsgSize:   4 << 20,
			KeepAliveTime:    time.Second,
			KeepAliveTimeout: time.Second,
		}
	}

	if err := m.Initialize(ctx, resolve); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	conn, err := m.GetConnection("health")
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}

	hc := grpc_health_v1.NewHealthClient(conn)
	resp, err := hc.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check rpc: %v", err)
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("want SERVING, got %v", resp.GetStatus())
	}

	// cчитает соединения serv
	if err := m.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	// соединение в шд
	if err := conn.Close(); err != nil {
		t.Fatalf("conn.Close: %v", err)
	}
	if err := m.HealthCheck(ctx); err == nil {
		t.Fatalf("want error because connection is SHUTDOWN")
	}

	// после полного Close() менеджера соединений быть не должно
	if err := m.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := m.GetConnection("health"); err == nil {
		t.Fatalf("want error after Close (connection removed)")
	}
}

func TestClient_AddGenericRegistration_PanicsOnBadConstructor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("want panic on invalid constructor")
		} else {
			_ = fmt.Sprintf("")
		}
	}()

	m := NewManager()

	// ожидаем func(grpc.ClientConnInterface) T, а возвращаем string
	bad := func(_ grpc.ClientConnInterface) string { return "nope" }
	AddGenericRegistration[int](m, "bad", bad)
}
