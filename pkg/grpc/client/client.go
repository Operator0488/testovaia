package client

import (
	"context"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/di"
	"git.vepay.dev/knoknok/backend-platform/pkg/grpc/client/interceptors"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"sync"
	"time"
)

// ClientRegistration репрезентует регистрацию gRPC клиента
type ClientRegistration struct {
	ServiceName string
	Address     string
	Constructor interface{} // func(grpc.ClientConnInterface) TClient
	Config      Config
}

// Config
type Config struct {
	Address          string
	Timeout          time.Duration
	MaxRecvMsgSize   int
	MaxSendMsgSize   int
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
}

type registration struct {
	serviceName string
	build       func(grpc.ClientConnInterface) any
	diRegister  func(ctx context.Context, inst any)
}

type Manager struct {
	mu                 sync.RWMutex
	connections        map[string]*grpc.ClientConn
	registrations      []registration
	unaryInterceptors  []grpc.UnaryClientInterceptor
	streamInterceptors []grpc.StreamClientInterceptor
}

func NewManager() *Manager {
	return &Manager{
		connections:        make(map[string]*grpc.ClientConn),
		unaryInterceptors:  make([]grpc.UnaryClientInterceptor, 0),
		streamInterceptors: make([]grpc.StreamClientInterceptor, 0),
	}
}

func AddGenericRegistration[T any](m *Manager, serviceName string, constructor any) {

	r, ok := constructor.(func(grpc.ClientConnInterface) T)
	if !ok {
		panic(fmt.Errorf("constructor for %q must be func(grpc.ClientConnInterface) %T", serviceName, *new(T)))
	}
	m.registrations = append(m.registrations, registration{
		serviceName: serviceName,
		build: func(conn grpc.ClientConnInterface) any {
			return r(conn)
		},
		diRegister: func(ctx context.Context, inst any) {
			typed := inst.(T)
			di.RegisterFactory[T](ctx, func() T { return typed })
			logger.Info(ctx, "gRPC client registered in DI",
				logger.String("service", serviceName),
			)
		},
	})
}

func (m *Manager) AddUnaryInterceptor(i grpc.UnaryClientInterceptor) {
	m.unaryInterceptors = append(m.unaryInterceptors, i)
}
func (m *Manager) AddStreamInterceptor(i grpc.StreamClientInterceptor) {
	m.streamInterceptors = append(m.streamInterceptors, i)
}

func (m *Manager) Initialize(ctx context.Context, resolveCfg func(service string) Config) error {
	unaryChain := []grpc.UnaryClientInterceptor{
		interceptors.MetricsUnaryInterceptor(),
	}
	unaryChain = append(unaryChain, m.unaryInterceptors...)

	streamChain := []grpc.StreamClientInterceptor{
		interceptors.MetricsStreamInterceptor(),
	}
	streamChain = append(streamChain, m.streamInterceptors...)

	for _, reg := range m.registrations {
		cfg := resolveCfg(reg.serviceName)
		if cfg.Address == "" {
			return fmt.Errorf("address not configured for service: %s", reg.serviceName)
		}

		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithStatsHandler(otelgrpc.NewClientHandler(
				otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
				otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
				otelgrpc.WithMessageEvents(otelgrpc.SentEvents, otelgrpc.ReceivedEvents),
			)),
			grpc.WithChainUnaryInterceptor(unaryChain...),
			grpc.WithChainStreamInterceptor(streamChain...),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(cfg.MaxRecvMsgSize),
				grpc.MaxCallSendMsgSize(cfg.MaxSendMsgSize),
			),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                cfg.KeepAliveTime,
				Timeout:             cfg.KeepAliveTimeout,
				PermitWithoutStream: true,
			}),
		}

		conn, err := grpc.NewClient(cfg.Address, opts...)
		if err != nil {
			return fmt.Errorf("failed to create connection for %s: %w", reg.serviceName, err)
		}

		m.mu.Lock()
		m.connections[reg.serviceName] = conn
		m.mu.Unlock()

		inst := reg.build(conn)
		reg.diRegister(ctx, inst)

		logger.Info(ctx, "gRPC client connection created",
			logger.String("service", reg.serviceName),
			logger.String("address", cfg.Address),
		)
	}
	return nil
}

func (m *Manager) GetConnection(service string) (*grpc.ClientConn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connections[service]
	if !ok {
		return nil, fmt.Errorf("connection not found for service: %s", service)
	}
	return conn, nil
}

func (m *Manager) Close() error {
	ctx := context.Background()
	logger.Info(ctx, "Closing gRPC client connections")

	m.mu.RLock()
	conns := make(map[string]*grpc.ClientConn, len(m.connections))
	for k, v := range m.connections {
		conns[k] = v
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(conns))
	for service, conn := range conns {
		go func(s string, c *grpc.ClientConn) {
			defer wg.Done()
			if err := c.Close(); err != nil {
				logger.Error(ctx, "Failed to close gRPC client connection", logger.String("service", s), logger.Err(err))
				return
			}
			logger.Info(ctx, "gRPC client connection closed", logger.String("service", s))
		}(service, conn)
	}
	wg.Wait()

	m.mu.Lock()
	m.connections = make(map[string]*grpc.ClientConn)
	m.mu.Unlock()
	return nil
}

func (m *Manager) HealthCheck(сtx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for service, conn := range m.connections {
		if conn.GetState().String() == "SHUTDOWN" {
			return fmt.Errorf("connection to %s is shutdown", service)
		}
	}
	return nil
}
