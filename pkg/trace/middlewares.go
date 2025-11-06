package trace

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/stats"
)

func MiddlewareEcho(serviceName string) echo.MiddlewareFunc {
	return echo.MiddlewareFunc(otelecho.Middleware(serviceName))
}

func ClientHandlerGRPC() stats.Handler {
	return otelgrpc.NewClientHandler()
}

func ServerHandlerGRPC() stats.Handler {
	return otelgrpc.NewServerHandler()
}
