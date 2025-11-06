package application

import (
	"context"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"git.vepay.dev/knoknok/backend-platform/pkg/swagger"
)

var (
	swaggerComponent = NewComponent("swagger", initSwagger, Noop)
)

// WithSwagger — включаем компонент
func WithSwagger(spec []byte) Option {
	return func(app *Application) error {
		app.swagger = swagger.New(spec)
		app.components.add(component(swaggerComponent))
		return nil
	}
}

func WithPublicSwaggerGateway(reg any, srv any) Option {
	return func(app *Application) error {
		if app.swagger == nil {
			return fmt.Errorf("call WithSwagger() before adding gateway registrars")
		}
		if app.PublicGrpcServer == nil {
			return fmt.Errorf("public gRPC server is not configured; swagger gateway is only allowed for public services")
		}

		// проверяем, является ли этот srv публичным
		if !app.PublicGrpcServer.CheckPublicServer(srv) {
			logger.Warn(context.Background(),
				"skip swagger gateway for non-public gRPC service",
				logger.String("reason", "service registered only on private server"),
			)
			return nil
		}

		app.swagger.Add(swagger.Gateway(reg, srv))
		return nil
	}
}

// WithSwaggerGatewayServer — RegisterXxxHandlerServer + srv.
func WithSwaggerGatewayServer(reg any, srv any) Option {
	return func(app *Application) error {
		if app.swagger == nil {
			return fmt.Errorf("call WithSwagger() before adding gateway registrars")
		}
		app.swagger.Add(swagger.Gateway(reg, srv))
		return nil
	}
}

func initSwagger(ctx context.Context, app *Application) error {
	mw, err := app.swagger.Middleware(ctx)
	if err != nil {
		return fmt.Errorf("swagger init failed: %w", err)
	}
	app.middlewares.Add(mw)
	return nil
}
