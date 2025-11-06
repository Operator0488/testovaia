package interceptors

import (
	"context"
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// MetricsUnaryInterceptor
func MetricsUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start).Seconds()
		statusCode := status.Code(err)

		metrics.GrpcClientRequestsTotal.WithLabelValues(method, statusCode.String()).Inc()
		metrics.GrpcClientRequestDuration.WithLabelValues(method).Observe(duration)

		return err
	}
}

// MetricsStreamInterceptor
func MetricsStreamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()

		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil {
			duration := time.Since(start).Seconds()
			statusCode := status.Code(err)
			metrics.GrpcClientRequestsTotal.WithLabelValues(method, statusCode.String()).Inc()
			metrics.GrpcClientRequestDuration.WithLabelValues(method).Observe(duration)
			return nil, err
		}

		return &metricsClientStream{
			ClientStream: clientStream,
			method:       method,
			start:        start,
		}, nil
	}
}

// metricsClientStream
type metricsClientStream struct {
	grpc.ClientStream
	method string
	start  time.Time
}

func (s *metricsClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		duration := time.Since(s.start).Seconds()
		statusCode := status.Code(err)
		metrics.GrpcClientRequestsTotal.WithLabelValues(s.method, statusCode.String()).Inc()
		metrics.GrpcClientRequestDuration.WithLabelValues(s.method).Observe(duration)
	}
	return err
}
