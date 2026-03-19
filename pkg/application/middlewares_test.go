package application

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "request validation failed", resp.Message)
	assert.NotEmpty(t, resp.Details)
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
