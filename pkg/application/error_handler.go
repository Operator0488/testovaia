package application

import (
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
)

type errorResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
	TraceID string   `json:"trace_id,omitempty"`
}

// WriteJSONError пишет JSON-ответ об ошибке и логирует его
func WriteJSONError(w http.ResponseWriter, r *http.Request, code int, message string, details ...string) {
	ctx := r.Context()

	var tid string
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.HasTraceID() {
		tid = sc.TraceID().String()
	}

	logger.Warn(ctx, "http error response",
		logger.Int("code", code),
		logger.String("message", message),
		logger.String("method", r.Method),
		logger.String("path", r.URL.Path),
	)

	resp := errorResponse{
		Code:    code,
		Message: message,
		Details: details,
		TraceID: tid,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(resp)
}
