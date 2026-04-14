package application

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/response"
)

var invalidSpec = []byte(`not: valid: yaml: [}`)

var minimalSpec = []byte(`
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    post:
      summary: Create user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - name
              properties:
                name:
                  type: string
      responses:
        "201":
          description: Created
  /users/{id}:
    get:
      summary: Get user
      parameters:
        - in: path
          name: id
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
`)

func TestNewMiddleware_ValidRequest(t *testing.T) {
	mw, err := httpValidationMiddleware(context.Background(), minimalSpec)
	require.NoError(t, err)

	called := false
	handler := mw(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	body := `{"name":"Bob"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)
	assert.True(t, called)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestNewMiddleware_MissingRequiredField(t *testing.T) {
	mw, err := httpValidationMiddleware(context.Background(), minimalSpec)
	require.NoError(t, err)

	handler := mw(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var env response.Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Contains(t, env.Error.Message, "request validation failed")
	assert.NotEmpty(t, env.Error.TraceID)
}

func TestNewMiddleware_InfraRoutesPassesThrough(t *testing.T) {
	mw, err := httpValidationMiddleware(context.Background(), minimalSpec)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		path       string
		shouldPass bool
	}{
		{
			name:       "metrics endpoint should bypass middleware",
			path:       "/metrics",
			shouldPass: true,
		},
		{
			name:       "healthz live endpoint should bypass middleware",
			path:       "/healthz/live",
			shouldPass: true,
		},
		{
			name:       "healthz ready endpoint should bypass middleware",
			path:       "/healthz/ready",
			shouldPass: true,
		},
		{
			name:       "swagger endpoint should bypass middleware",
			path:       "/swagger",
			shouldPass: true,
		},
		{
			name:       "swagger openapi yaml should bypass middleware",
			path:       "/swagger/openapi.yaml",
			shouldPass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := mw(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if tc.shouldPass {
				assert.True(t, called, "Handler should be called for path: %s", tc.path)
			} else {
				assert.False(t, called, "Handler should not be called for path: %s", tc.path)
			}
		})
	}
}

func newTestApp() *Application {
	return &Application{
		auth:      &authConfig{},
		rateLimit: &rateLimitConfig{},
	}
}

func TestAuthMiddleware_InfraPathsBypassAuthFunc(t *testing.T) {
	bypassedPaths := []string{
		"/healthz/live",
		"/healthz/ready",
		"/metrics",
		"/swagger/index.html",
	}

	for _, path := range bypassedPaths {
		t.Run(path, func(t *testing.T) {
			app := newTestApp()
			called := false
			app.auth.fn = func(r *http.Request) (*http.Request, error) {
				called = true
				return r, nil
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			app.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})(rec, req)

			assert.False(t, called, "authFunc не должна вызываться для пути %s", path)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestAuthMiddleware_NilAuthFuncPassesThrough(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	rec := httptest.NewRecorder()
	called := false

	app.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_Returns401WhenAuthFuncErrors(t *testing.T) {
	app := newTestApp()
	app.auth.fn = func(r *http.Request) (*http.Request, error) {
		return nil, errors.New("invalid token")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	rec := httptest.NewRecorder()

	app.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_EnrichedRequestPassedToHandler(t *testing.T) {
	app := newTestApp()
	type ctxKey string
	const key ctxKey = "claims"

	app.auth.fn = func(r *http.Request) (*http.Request, error) {
		ctx := context.WithValue(r.Context(), key, "injected-claims")
		return r.WithContext(ctx), nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	rec := httptest.NewRecorder()
	var gotValue string

	app.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		gotValue, _ = r.Context().Value(key).(string)
		w.WriteHeader(http.StatusOK)
	})(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "injected-claims", gotValue)
}

func TestNewMiddleware_InvalidSpec(t *testing.T) {
	_, err := httpValidationMiddleware(context.Background(), invalidSpec)
	assert.Error(t, err)
}

func TestNewMiddleware_BodyRestoredForHandler(t *testing.T) {
	mw, err := httpValidationMiddleware(context.Background(), minimalSpec)
	require.NoError(t, err)

	var receivedBody string
	handler := mw(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(http.StatusCreated)
	})

	body := `{"name":"Charlie"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, body, receivedBody, "request body must be intact for the downstream handler")
}
