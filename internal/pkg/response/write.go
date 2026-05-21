package response

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// WriteError — пишет ошибку в ответ клиенту и логирует ее (если необходимо)
func WriteError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	writeError(ctx, w, r.Method, r.URL.Path, err)
}

// WriteJSON — тело успешного ответа (тип из OpenAPI).
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// JSONErrorHandler — для oapi-codegen ErrorHandlerFunc.
func JSONErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	WriteError(r.Context(), w, r, err)
}

func writeError(ctx context.Context, w http.ResponseWriter, method, path string, err error) {
	var httpErr HTTPError
	if !errors.As(err, &httpErr) {
		httpErr = Internal(err)
	}

	var serverErr *serverError
	if errors.As(httpErr, &serverErr) && serverErr.cause != nil {
		logger.Error(ctx, "http error",
			logger.Int("http_status", httpErr.HTTPStatus()),
			logger.String("code", httpErr.Code()),
			logger.String("cause", serverErr.cause.Error()),
			logger.String("method", method),
			logger.String("path", path),
		)
	}

	errPart := &ErrorPart{
		TraceID: traceIDFromContext(ctx),
		Message: httpErr.Message(),
	}

	var valErr *validationError
	if errors.As(httpErr, &valErr) {
		errPart.Detail = valErr.Details()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpErr.HTTPStatus())
	_ = json.NewEncoder(w).Encode(Envelope{
		Error: errPart,
	})
}

func traceIDFromContext(ctx context.Context) string {
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.HasTraceID() {
		return sc.TraceID().String()
	}

	return uuid.New().String()
}
