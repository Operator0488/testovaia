package application

import (
	"context"
	"fmt"
	grpcserver "git.vepay.dev/knoknok/backend-platform/pkg/grpc/server"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"google.golang.org/grpc"
)

var (
	grpcPrivateServerComponent = NewComponent("grpc-private-server", initPrivateGrpcServer, runPrivateGrpcServer)
)

// helper
func (a *Application) addPrivateGrpcServer() {
	if a.PrivateGrpcServer != nil {
		return
	}
	cfg := a.config.GetGrpcPrivateServerConfig()
	a.PrivateGrpcServer = grpcserver.New(grpcserver.Config{
		Addr:              cfg.GetAddr(),
		MaxRecvMsgSize:    cfg.MaxRecvMsgSize,
		MaxSendMsgSize:    cfg.MaxSendMsgSize,
		ConnectionTimeout: cfg.ConnectionTimeout,
		KeepAliveTime:     cfg.KeepAliveTime,
		KeepAliveTimeout:  cfg.KeepAliveTimeout,
	})
}

func WithPrivateGrpcServer[T any](register func(s grpc.ServiceRegistrar, srv T), instance T) Option {
	return func(app *Application) error {
		app.components.add(component(grpcPrivateServerComponent))
		app.addPrivateGrpcServer()
		app.PrivateGrpcServer.AddService(func(sr grpc.ServiceRegistrar) {
			register(sr, instance)
		})
		return nil
	}
}

func WithPrivateGrpcUnaryInterceptor(i grpc.UnaryServerInterceptor) Option {
	return func(app *Application) error {
		app.PrivateGrpcServer.AddUnaryInterceptor(i)
		return nil
	}
}
func WithPrivateGrpcStreamInterceptor(i grpc.StreamServerInterceptor) Option {
	return func(app *Application) error {
		app.PrivateGrpcServer.AddStreamInterceptor(i)
		return nil
	}
}

func initPrivateGrpcServer(ctx context.Context, app *Application) error {
	if app.PrivateGrpcServer == nil {
		return fmt.Errorf("private gRPC server not configured (no services were registered)")
	}

	if app.PublicGrpcServer != nil && app.PublicGrpcServer.Addr() == app.PrivateGrpcServer.Addr() {
		return fmt.Errorf("private/public gRPC servers must listen on different addresses: both set to %s",
			app.PrivateGrpcServer.Addr())
	}

	logger.Info(ctx, "private gRPC server initialize",
		logger.String("addr", app.PrivateGrpcServer.Addr()),
		logger.Int("services", app.PrivateGrpcServer.ServicesCount()),
	)
	if err := app.PrivateGrpcServer.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize private gRPC server: %w", err)
	}
	app.Closer.Add(app.PrivateGrpcServer.Stop)
	return nil
}

func runPrivateGrpcServer(ctx context.Context, app *Application) error {
	if app.PrivateGrpcServer == nil {
		return fmt.Errorf("private gRPC server not initialized")
	}
	if err := app.PrivateGrpcServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start private gRPC server: %w", err)
	}
	app.Health.Add("grpc-server-private", app.PrivateGrpcServer.HealthCheck)
	logger.Info(ctx, "private gRPC server started")
	return nil
}
