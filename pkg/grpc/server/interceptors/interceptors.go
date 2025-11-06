package interceptors

import (
	"context"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	"runtime/debug"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsUnaryInterceptor
func MetricsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start).Seconds()
		statusCode := status.Code(err)

		metrics.GrpcServerRequestsTotal.WithLabelValues(info.FullMethod, statusCode.String()).Inc()
		metrics.GrpcServerRequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

		return resp, err
	}
}

// MetricsStreamInterceptor
func MetricsStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start).Seconds()
		statusCode := status.Code(err)

		metrics.GrpcServerRequestsTotal.WithLabelValues(info.FullMethod, statusCode.String()).Inc()
		metrics.GrpcServerRequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

		return err
	}
}

// RecoveryUnaryInterceptor рекавери
func RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.Error(ctx, "gRPC panic recovered",
					logger.String("method", info.FullMethod),
					logger.String("panic", fmt.Sprintf("%v", r)),
					logger.String("stack", string(stack)),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// RecoveryStreamInterceptor рекавери
func RecoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.Error(ss.Context(), "gRPC stream panic recovered",
					logger.String("method", info.FullMethod),
					logger.String("panic", fmt.Sprintf("%v", r)),
					logger.String("stack", string(stack)),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}
