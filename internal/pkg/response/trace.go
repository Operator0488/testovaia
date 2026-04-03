package response

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

func traceIDFromContext(ctx context.Context) string {
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.HasTraceID() {
		return sc.TraceID().String()
	}

	return uuid.New().String()
}
