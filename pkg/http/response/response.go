package response

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
)

// Envelope — единая структура JSON-ответа платформы
// Успех: payload содержит DTO, error = null
// Ошибка: payload = null, error содержит traceId и message
type Envelope struct {
	Payload any        `json:"payload,omitempty"`
	Error   *ErrorPart `json:"error,omitempty"`
}

// ErrorPart — часть ответа при ошибке
type ErrorPart struct {
	TraceID string `json:"traceId"`
	Message string `json:"message"`
}

// WriteError пишет JSON-ответ об ошибке в формате Envelope и логирует его
func WriteError(w http.ResponseWriter, r *http.Request, code int, message string) {
	ctx := r.Context()
	tid := traceID(ctx)

	logger.Warn(ctx, "http error response",
		logger.Int("code", code),
		logger.String("message", message),
		logger.String("method", r.Method),
		logger.String("path", r.URL.Path),
	)

	env := Envelope{Error: &ErrorPart{TraceID: tid, Message: message}}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(env)
}

func traceID(ctx context.Context) string {
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.HasTraceID() {
		return sc.TraceID().String()
	}

	return uuid.New().String()
}
