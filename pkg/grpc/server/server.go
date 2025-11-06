package server

import (
	"context"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/grpc/server/interceptors"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"net"
	"reflect"
	"sync"
	"time"
)

const (
	timestop = 10 * time.Second //время для ожидания выполнения m.grpc.GracefulStop()
)

// Конфиг сервера
type Config struct {
	Addr              string
	MaxRecvMsgSize    int
	MaxSendMsgSize    int
	ConnectionTimeout time.Duration
	KeepAliveTime     time.Duration
	KeepAliveTimeout  time.Duration
}

// манагер сервера
type Manager struct {
	cfg Config

	addr          string
	listener      net.Listener
	grpc          *grpc.Server
	health        *health.Server
	mu            sync.Mutex
	registrations []func(grpc.ServiceRegistrar)

	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor

	publicServices map[uintptr]struct{}
}

func New(cfg Config) *Manager {
	return &Manager{
		cfg:                cfg,
		addr:               cfg.Addr,
		registrations:      make([]func(grpc.ServiceRegistrar), 0),
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
		publicServices:     make(map[uintptr]struct{}),
	}
}

func (m *Manager) Addr() string       { return m.addr }
func (m *Manager) ServicesCount() int { return len(m.registrations) }

func (m *Manager) AddUnaryInterceptor(in grpc.UnaryServerInterceptor) {
	m.unaryInterceptors = append(m.unaryInterceptors, in)
}

func (m *Manager) AddStreamInterceptor(in grpc.StreamServerInterceptor) {
	m.streamInterceptors = append(m.streamInterceptors, in)
}

func (m *Manager) AddService(register func(grpc.ServiceRegistrar)) {
	m.mu.Lock()
	m.registrations = append(m.registrations, register)
	m.mu.Unlock()
}

func (m *Manager) Initialize(ctx context.Context) error {
	// Базовые цепочки: recovery + metrics.
	unaryChain := []grpc.UnaryServerInterceptor{
		interceptors.RecoveryUnaryInterceptor(),
		interceptors.MetricsUnaryInterceptor(),
	}
	unaryChain = append(unaryChain, m.unaryInterceptors...)

	streamChain := []grpc.StreamServerInterceptor{
		interceptors.RecoveryStreamInterceptor(),
		interceptors.MetricsStreamInterceptor(),
	}
	streamChain = append(streamChain, m.streamInterceptors...)

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryChain...),
		grpc.ChainStreamInterceptor(streamChain...),
		grpc.MaxRecvMsgSize(m.cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(m.cfg.MaxSendMsgSize),
		grpc.ConnectionTimeout(m.cfg.ConnectionTimeout),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    m.cfg.KeepAliveTime,
			Timeout: m.cfg.KeepAliveTimeout,
		}),

		grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
			otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
			otelgrpc.WithMessageEvents(otelgrpc.SentEvents, otelgrpc.ReceivedEvents),
		)),
	}

	m.grpc = grpc.NewServer(opts...)

	// health
	m.health = health.NewServer()
	grpc_health_v1.RegisterHealthServer(m.grpc, m.health)

	// регистрируем добавленные сервисы
	for _, reg := range m.registrations {
		reg(m.grpc)
	}

	// общая health
	m.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	logger.Info(ctx, "gRPC services registered",
		logger.Int("count", len(m.registrations)),
	)

	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	l, err := net.Listen("tcp", m.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", m.addr, err)
	}
	m.listener = l

	logger.Info(ctx, "gRPC server starting", logger.String("addr", m.addr))

	go func() {
		if err := m.grpc.Serve(l); err != nil {
			logger.Error(ctx, "gRPC server error", logger.Err(err))
		}
	}()
	return nil
}

func (m *Manager) Stop() error {
	ctx := context.Background()
	logger.Info(ctx, "gRPC server stopping")

	if m.health != nil {
		m.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}

	stopped := make(chan struct{})
	go func() {
		m.grpc.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		logger.Info(ctx, "gRPC server stopped gracefully")
	case <-time.After(timestop):
		logger.Warn(ctx, "gRPC server stop timeout, forcing shutdown")
		m.grpc.Stop()
	}
	return nil
}

func (m *Manager) HealthCheck(ctx context.Context) error {
	if m.grpc == nil {
		return fmt.Errorf("gRPC server not initialized")
	}
	if m.listener == nil {
		return fmt.Errorf("gRPC server not started")
	}
	return nil
}

func (m *Manager) MarkPublicService(srv any) {
	ptr := reflect.ValueOf(srv).Pointer()
	m.mu.Lock()
	m.publicServices[ptr] = struct{}{}
	m.mu.Unlock()
}

func (m *Manager) CheckPublicServer(srv any) bool {
	ptr := reflect.ValueOf(srv).Pointer()
	m.mu.Lock()
	_, ok := m.publicServices[ptr]
	m.mu.Unlock()
	return ok
}
