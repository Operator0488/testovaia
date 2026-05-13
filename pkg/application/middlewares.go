package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/middleware"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/response"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	pkgauth "easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/auth"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/metrics"
)

const maxBodyBytes = 1 << 20

func noopHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}
}

// Health

type HealthResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Code      int       `json:"code"`
}

func checkReadiness(ctx context.Context, a *Application) HealthResponse {
	response := HealthResponse{
		Timestamp: time.Now().UTC(),
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
				Timestamp: time.Now().UTC(),
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
	tracer := otel.Tracer(fmt.Sprintf("%s/http", a.config.GetAppName()))
	propagator := otel.GetTextMapPropagator()

	return func(w http.ResponseWriter, r *http.Request) {
		// не считаем метрики для metrics и healthz
		if isMetricPath(r.URL.Path) {
			next(w, r)

			return
		}

		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			oteltrace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLPath(r.URL.Path),
				semconv.ServerAddress(r.Host),
			),
		)
		defer span.End()

		start := time.Now().UTC()
		rec := &statusRecorder{ResponseWriter: w}
		next(rec, r.WithContext(ctx))

		// если хендлер вообще ничего не писал - 200
		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		method := r.Method
		path := normalizePath(r)
		status := strconv.Itoa(rec.status)
		duration := time.Since(start).Seconds()

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDurationSeconds.WithLabelValues(method, path, status).Observe(duration)

		span.SetName(fmt.Sprintf("%s %s", method, path))
		span.SetAttributes(semconv.HTTPResponseStatusCode(rec.status))
	}
}

// Panic Recovery

func (a *Application) panicRecoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				ctx := r.Context()

				logger.Error(ctx, "panic recovered",
					logger.String("method", r.Method),
					logger.String("path", r.URL.Path),
					logger.String("panic", fmt.Sprintf("%v", rec)),
					logger.String("stack", string(debug.Stack())),
				)

				var err error
				switch v := rec.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("panic: %v", v)
				}

				response.WriteError(ctx, w, r, response.Internal(err))
			}
		}()
		next(w, r)
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

type authFunc func(r *http.Request) (*http.Request, error)

type authConfig struct {
	mu       sync.RWMutex
	fn       authFunc
	warnOnce sync.Once
}

// Auth

// authMiddleware создает middleware для проверки аутентификации запросов
//
// Middleware работает по следующему принципу:
//  1. Пропускает запросы к служебным путям без проверки (/healthz/*, /metrics, /swagger*)
//  2. Если маршрут найден в OpenAPI-спецификации и не требует auth (security: []) — пропускает
//  3. Если функция аутентификации не настроена (fn == nil) — предупреждение и пропуск
//  4. После успешной аутентификации проверяет scope токена по требованиям из спецификации
func (a *Application) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if middleware.IsInfraPath(r.URL.Path) {
			next(w, r)

			return
		}

		secReqs, authRequired := routeSecurityRequirements(r, a.openAPIRouters)
		if !authRequired {
			next(w, r)

			return
		}

		ctx := r.Context()

		a.auth.mu.RLock()
		fn := a.auth.fn
		a.auth.mu.RUnlock()

		if fn == nil {
			a.auth.warnOnce.Do(func() {
				logger.Warn(r.Context(),
					"authentication middleware is not configured, all requests are allowed",
				)
			})
			next(w, r)

			return
		}

		newReq, err := fn(r)
		if err != nil {
			response.WriteError(ctx, w, r, response.Unauthorized())

			return
		}

		if hasScopes(secReqs) {
			claims := pkgauth.ClaimsFromContext(newReq.Context())
			if claims == nil || !scopesSatisfied(claims.Scopes, secReqs) {
				response.WriteError(ctx, w, r, response.Forbidden("недостаточно прав доступа"))

				return
			}
		}

		next(w, newReq)
	}
}

// routeSecurityRequirements возвращает требования безопасности для маршрута и флаг необходимости аутентификации.
// Возвращает (nil, false) для публичных маршрутов, (reqs, true) — если аутентификация нужна.
// Если маршрут не найден ни в одной спецификации — fail-secure: (nil, true).
// Operation-level security переопределяет doc-level security.
func routeSecurityRequirements(r *http.Request, routerList []routers.Router) (openapi3.SecurityRequirements, bool) {
	for _, router := range routerList {
		route, _, err := router.FindRoute(r)
		if err != nil {
			continue
		}

		// operation-level security переопределяет doc-level
		if route.Operation.Security != nil {
			if len(*route.Operation.Security) == 0 {
				return nil, false
			}

			return *route.Operation.Security, true
		}

		// fallback на doc-level security
		if len(route.Spec.Security) == 0 {
			return nil, false
		}

		return route.Spec.Security, true
	}

	// маршрут не описан в спеке — считаем защищённым
	return nil, true
}

// hasScopes возвращает true, если хотя бы одно требование содержит непустой список scopes.
func hasScopes(reqs openapi3.SecurityRequirements) bool {
	for _, req := range reqs {
		for _, scopes := range req {
			if len(scopes) > 0 {
				return true
			}
		}
	}

	return false
}

// scopesSatisfied проверяет, удовлетворяет ли scope токена хотя бы одному требованию из списка.
// SecurityRequirements (массив) — OR: достаточно выполнить хотя бы одно требование.
// Внутри SecurityRequirement — AND: токен обязан иметь все scopes из каждой схемы.
func scopesSatisfied(tokenScopes []pkgauth.Scope, reqs openapi3.SecurityRequirements) bool {
	for _, req := range reqs {
		if requirementSatisfied(tokenScopes, req) {
			return true
		}
	}

	return false
}

// requirementSatisfied проверяет, что все scopes из требовании пристутствуют в предъявленом токене (AND-семантика).
func requirementSatisfied(tokenScopes []pkgauth.Scope, req openapi3.SecurityRequirement) bool {
	for _, reqScopes := range req {
		for _, reqScope := range reqScopes {
			reqScopeFound := false
			for _, tokenScope := range tokenScopes {
				if pkgauth.Scope(reqScope) == tokenScope {
					reqScopeFound = true
					break
				}
			}
			if !reqScopeFound {
				return false
			}
		}
	}

	return true
}

// Rate limit

type rateLimitFunc func(r *http.Request) error

type rateLimitConfig struct {
	mu       sync.Mutex
	fn       rateLimitFunc
	warnOnce sync.Once
}

// rateLimitMiddleware создает middleware для ограничения частоты запросов
//
// Middleware работает по следующему принципу:
//  1. Пропускает запросы к служебным путям без проверки:
//     - /healthz/live
//     - /healthz/ready
//     - /metrics
//     - Любые пути, начинающиеся с /swagger
//  2. Если функция rate limit не настроена (fn == nil) - временная заглушка:
//     - Выводит предупреждение в лог (один раз за время работы приложения)
//     - Пропускает запрос без проверки
func (a *Application) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if middleware.IsInfraPath(r.URL.Path) {
			next(w, r)

			return
		}

		ctx := r.Context()

		a.rateLimit.mu.Lock()
		fn := a.rateLimit.fn
		a.rateLimit.mu.Unlock()

		if fn == nil {
			a.rateLimit.warnOnce.Do(func() {
				logger.Warn(ctx,
					"rate limit middleware is not configured, all requests are allowed",
				)
			})
			next(w, r)

			return
		}

		if err := fn(r); err != nil {
			response.WriteError(ctx, w, r, response.TooManyRequests())

			return
		}

		next(w, r)
	}
}

// OpenAPI

// buildRouters парсит байты спецификаций и строит список kin-openapi роутеров.
func buildRouters(ctx context.Context, specs [][]byte) ([]routers.Router, error) {
	routerList := make([]routers.Router, 0, len(specs))
	for _, spec := range specs {
		r, err := newRouter(ctx, spec)
		if err != nil {
			return nil, err
		}
		routerList = append(routerList, r)
	}

	return routerList, nil
}

// httpValidationMiddleware возвращает стандартный net/http middleware, который валидирует
// входящие запросы на соответствие одной или нескольким OpenAPI спецификациям.
// Маршруты, не описанные ни в одной спецификации (например, /healthz, /metrics),
// пропускаются без валидации — не зависит от фреймворка, работает с echo, gin и др.
//
// При передаче нескольких спецификаций middleware последовательно ищет маршрут
// в каждом роутере: если маршрут найден — валидирует запрос, если нет —
// переходит к следующей спецификации.
func httpValidationMiddleware(ctx context.Context, routerList []routers.Router) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := validateMultiRequest(ctx, r, routerList); err != nil {
				response.WriteError(ctx, w, r, response.BadRequest(
					fmt.Sprintf("request validation failed: %s", getReason(err))))

				return
			}
			next(w, r)
		}
	}
}

// getReason возвращает причину ошибки валидации запроса
func getReason(err error) string {
	var validationErr *openapi3filter.RequestError
	if errors.As(err, &validationErr) && validationErr.Err != nil {
		var schemaErr *openapi3.SchemaError
		if errors.As(validationErr.Err, &schemaErr) {
			return schemaErr.Reason
		}

		return validationErr.Err.Error()
	}

	return err.Error()
}

// newRouter парсит байты спецификации и строит kin-openapi роутер.
func newRouter(ctx context.Context, spec []byte) (routers.Router, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(spec)
	if err != nil {
		return nil, fmt.Errorf("openapi: failed to load OpenAPI spec: %w", err)
	}

	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("openapi: invalid OpenAPI spec: %w", err)
	}

	r, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("openapi: failed to create router: %w", err)
	}

	return r, nil
}

// validateMultiRequest ищет маршрут запроса среди всех роутеров.
// Если маршрут найден в одном из них — валидирует. Если не найден ни в одном — пропускает
func validateMultiRequest(ctx context.Context, r *http.Request, routers []routers.Router) error {
	if middleware.IsInfraPath(r.URL.Path) {
		return nil
	}

	var bodyBuf []byte
	if r.Body != nil {
		var err error
		bodyBuf, err = io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
		if err != nil {
			return fmt.Errorf("openapi: failed to read request body: %w", err)
		}
		restoreBody(r, bodyBuf)
	}

	for _, router := range routers {
		route, pathParams, err := router.FindRoute(r)
		if err != nil {
			// пропускаем, если маршрут не найден
			continue
		}

		input := &openapi3filter.RequestValidationInput{
			Request:    r,
			PathParams: pathParams,
			Route:      route,
			Options: &openapi3filter.Options{
				AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			},
		}

		if err := openapi3filter.ValidateRequest(ctx, input); err != nil {
			return err
		}

		restoreBody(r, bodyBuf)

		return nil
	}

	return nil
}

func restoreBody(r *http.Request, buf []byte) {
	if buf != nil {
		r.Body = io.NopCloser(bytes.NewReader(buf))
	}
}
