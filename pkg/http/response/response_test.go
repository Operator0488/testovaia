package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	WriteError(rec, req, http.StatusUnauthorized, "unauthorized")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "unauthorized", env.Error.Message)
	assert.NotEmpty(t, env.Error.TraceID)
	assert.Nil(t, env.Payload)
}

func TestEnvelopeMiddleware_WrapsSuccess(t *testing.T) {
	handler := EnvelopeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Nil(t, env.Error)
	require.NotNil(t, env.Payload)
	p, ok := env.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "123", p["id"])
}

func TestEnvelopeMiddleware_PassesThroughInfra(t *testing.T) {
	handler := EnvelopeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Не обёрнуто в envelope
	assert.Equal(t, `{"status":"ok"}`, rec.Body.String())
}

func TestEnvelopeMiddleware_PassesThroughError(t *testing.T) {
	handler := EnvelopeMiddleware(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusForbidden, "forbidden")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "forbidden", env.Error.Message)
}
