package application

import (
	"context"
	"fmt"
	grpcserver "git.vepay.dev/knoknok/backend-platform/pkg/grpc/server"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"google.golang.org/grpc"
)

var (
	grpcPublicServerComponent = NewComponent("grpc-public-server", initPublicGrpcServer, runPublicGrpcServer)
)

func (a *Application) addPublicGrpcServer() {
	if a.PublicGrpcServer != nil {
		return
	}
	cfg := a.config.GetGrpcPublicServerConfig()
	a.PublicGrpcServer = grpcserver.New(grpcserver.Config{
		Addr:              cfg.GetAddr(),
		MaxRecvMsgSize:    cfg.MaxRecvMsgSize,
		MaxSendMsgSize:    cfg.MaxSendMsgSize,
		ConnectionTimeout: cfg.ConnectionTimeout,
		KeepAliveTime:     cfg.KeepAliveTime,
		KeepAliveTimeout:  cfg.KeepAliveTimeout,
	})
}

func WithPublicGrpcServer[T any](register func(s grpc.ServiceRegistrar, srv T), instance T) Option {
	return func(app *Application) error {
		app.components.add(component(grpcPublicServerComponent))
		app.addPublicGrpcServer()
		app.PublicGrpcServer.MarkPublicService(instance)
		app.PublicGrpcServer.AddService(func(sr grpc.ServiceRegistrar) {
			register(sr, instance)
		})
		return nil
	}
}

func WithPublicGrpcUnaryInterceptor(i grpc.UnaryServerInterceptor) Option {
	return func(app *Application) error {
		app.PublicGrpcServer.AddUnaryInterceptor(i)
		return nil
	}
}
func WithPublicGrpcStreamInterceptor(i grpc.StreamServerInterceptor) Option {
	return func(app *Application) error {
		app.PublicGrpcServer.AddStreamInterceptor(i)
		return nil
	}
}

func initPublicGrpcServer(ctx context.Context, app *Application) error {
	if app.PublicGrpcServer == nil {
		return fmt.Errorf("public gRPC server not configured (no services were registered)")
	}
	if app.PrivateGrpcServer != nil && app.PublicGrpcServer.Addr() == app.PrivateGrpcServer.Addr() {
		return fmt.Errorf("private/public gRPC servers must listen on different addresses: both set to %s",
			app.PublicGrpcServer.Addr())
	}
	logger.Info(ctx, "public gRPC server initialize",
		logger.String("addr", app.PublicGrpcServer.Addr()),
		logger.Int("services", app.PublicGrpcServer.ServicesCount()),
	)
	if err := app.PublicGrpcServer.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize public gRPC server: %w", err)
	}
	app.Closer.Add(app.PublicGrpcServer.Stop)
	return nil
}

func runPublicGrpcServer(ctx context.Context, app *Application) error {
	if app.PublicGrpcServer == nil {
		return fmt.Errorf("public gRPC server not initialized")
	}
	if err := app.PublicGrpcServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start public gRPC server: %w", err)
	}
	app.Health.Add("grpc-server-public", app.PublicGrpcServer.HealthCheck)
	logger.Info(ctx, "public gRPC server started")
	return nil
}
