package application

import (
	"context"
	"encoding/json"
	"git.vepay.dev/knoknok/backend-platform/pkg/metrics"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

func noopHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Code      int       `json:"code"`
}

func checkReadiness(ctx context.Context, a *Application) HealthResponse {
	response := HealthResponse{
		Timestamp: time.Now(),
		Code:      http.StatusServiceUnavailable,
	}

	// app is shutting down
	select {
	case <-a.closing:
		response.Status = "not_ready"
		response.Message = "Application is shutting down"
		return response
	default:
	}

	select {
	case <-a.started:
		// app was started, check health
		if err := a.Health.Check(ctx); err != nil {
			logger.Error(ctx, "Application got health check error", logger.Err(err))
			response.Status = "not_ready"
			response.Message = "Application has problems"
			response.Code = http.StatusServiceUnavailable
		} else {
			response.Status = "ready"
			response.Message = "Application is ready to accept requests"
			response.Code = http.StatusOK
		}
	case <-ctx.Done():
		response.Status = "not_ready"
		response.Message = "Application is still starting up"
	}

	return response
}

// livenessMiddleware return application is alive
func (a *Application) livenessMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz/live" {
			w.Header().Set("Content-Type", "application/json")
			response := HealthResponse{
				Timestamp: time.Now(),
				Status:    "healthy",
				Message:   "Application is running",
				Code:      http.StatusOK,
			}
			w.WriteHeader(response.Code)
			json.NewEncoder(w).Encode(response)
			return
		}
		next(w, r)
	}
}

// readinessMiddleware return application is started, healthcheck return success
func (a *Application) readinessMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz/ready" {
			w.Header().Set("Content-Type", "application/json")
			ctx, cancel := context.WithTimeout(r.Context(), defaultReadinessTimeout)
			defer cancel()
			response := checkReadiness(ctx, a)
			w.WriteHeader(response.Code)
			json.NewEncoder(w).Encode(response)
			return
		}
		next(w, r)
	}
}

// Metrics

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (a *Application) httpMetricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// не считаем метрики для metrics и healthz
		if isMetricPath(r.URL.Path) {
			next(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next(rec, r)

		// TODO если хендлер вообще ничего не писал то 200
		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		method := r.Method
		path := normalizePath(r)
		status := strconv.Itoa(rec.status)
		t := time.Since(start).Seconds()

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDurationSeconds.WithLabelValues(method, path, status).Observe(t)
	}
}

func isMetricPath(p string) bool {
	return p == "/metrics" || strings.HasPrefix(p, "/healthz/")
}

func normalizePath(r *http.Request) string {
	// TODO для нормализации параметров
	return r.URL.Path
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	// если WriteHeader ещё не вызывали — считаем 200
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
