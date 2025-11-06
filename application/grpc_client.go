package application

import (
	"context"
	"fmt"
	grpc1 "git.vepay.dev/knoknok/backend-platform/pkg/grpc"
	"git.vepay.dev/knoknok/backend-platform/pkg/grpc/client"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

var (
	grpcClientComponent = NewComponent("grpc-client", initGrpcClient, Noop)
)

// WithGrpcClient
// constructor - proto-конструктор (например, userv2.NewUserServiceClient).
func WithGrpcClient[TClient any](serviceName string, constructor any) Option {
	return func(app *Application) error {

		if app.GrpcClients == nil {
			app.GrpcClients = client.NewManager()
			app.components.add(component(grpcClientComponent))
		}

		client.AddGenericRegistration[TClient](app.GrpcClients, serviceName, constructor)
		return nil
	}
}

func initGrpcClient(ctx context.Context, app *Application) error {
	if app.GrpcClients == nil {
		return fmt.Errorf("gRPC client manager not created")
	}

	grpc1.EnableWithContext(ctx)

	resolveCfg := func(serviceName string) client.Config {
		cfg := app.config.GetGrpcClientConfig(serviceName)
		return client.Config{
			Address:          cfg.Address,
			Timeout:          cfg.Timeout,
			MaxRecvMsgSize:   cfg.MaxRecvMsgSize,
			MaxSendMsgSize:   cfg.MaxSendMsgSize,
			KeepAliveTime:    cfg.KeepAliveTime,
			KeepAliveTimeout: cfg.KeepAliveTimeout,
		}
	}

	logger.Info(ctx, "gRPC clients initialize")
	if err := app.GrpcClients.Initialize(ctx, resolveCfg); err != nil {
		return fmt.Errorf("failed to initialize gRPC clients: %w", err)
	}

	app.Health.Add("grpc-clients", app.GrpcClients.HealthCheck)
	app.Closer.Add(app.GrpcClients.Close)
	return nil
}
