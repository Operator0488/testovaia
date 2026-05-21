package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteError_ClientErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        HTTPError
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "NotFound",
			err:        NotFound(),
			wantStatus: http.StatusNotFound,
			wantMsg:    "resource not found",
		},
		{
			name:       "BadRequest",
			err:        BadRequest("invalid input"),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid input",
		},
		{
			name:       "Conflict",
			err:        Conflict("already exists"),
			wantStatus: http.StatusConflict,
			wantMsg:    "already exists",
		},
		{
			name:       "Forbidden",
			err:        Forbidden("access denied"),
			wantStatus: http.StatusForbidden,
			wantMsg:    "access denied",
		},
		{
			name:       "Unauthorized",
			err:        Unauthorized(),
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "Authentication required",
		},
		{
			name:       "Locked",
			err:        Locked("resource locked"),
			wantStatus: http.StatusLocked,
			wantMsg:    "resource locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			WriteError(r.Context(), w, r, tt.err)

			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.Equal(t, tt.wantStatus, w.Code)

			var env Envelope
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
			require.NotNil(t, env.Error)
			assert.Equal(t, tt.wantMsg, env.Error.Message)
			assert.NotEmpty(t, env.Error.TraceID)
			assert.Nil(t, env.Error.Detail)
			assert.Nil(t, env.Payload)
		})
	}
}

func TestWriteError_ServerErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        HTTPError
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "Internal",
			err:        Internal(errors.New("db connection failed")),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "An internal error occurred",
		},
		{
			name:       "TooManyRequests",
			err:        TooManyRequests(),
			wantStatus: http.StatusTooManyRequests,
			wantMsg:    "Too many requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			WriteError(r.Context(), w, r, tt.err)

			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.Equal(t, tt.wantStatus, w.Code)

			var env Envelope
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
			require.NotNil(t, env.Error)
			assert.Equal(t, tt.wantMsg, env.Error.Message)
			assert.NotEmpty(t, env.Error.TraceID)
			assert.Nil(t, env.Error.Detail)
			assert.Nil(t, env.Payload)
		})
	}
}

func TestWriteError_ValidationError(t *testing.T) {
	t.Run("with details", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/resource", nil)

		details := map[string][]string{
			"name":  {"name is required"},
			"email": {"invalid format", "must be unique"},
		}
		err := UnprocessableEntity(details)
		WriteError(r.Context(), w, r, err)

		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var env Envelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
		require.NotNil(t, env.Error)
		assert.Equal(t, Message, env.Error.Message)
		assert.NotEmpty(t, env.Error.TraceID)
		require.NotNil(t, env.Error.Detail)
		assert.Equal(t, details, env.Error.Detail)
		assert.Nil(t, env.Payload)
	})

	t.Run("without details (nil)", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/resource", nil)

		err := UnprocessableEntity(nil)
		WriteError(r.Context(), w, r, err)

		var env Envelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
		require.NotNil(t, env.Error)
		assert.Equal(t, Message, env.Error.Message)
		assert.NotEmpty(t, env.Error.TraceID)
		assert.Nil(t, env.Error.Detail)
	})

	t.Run("without details (empty map)", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/resource", nil)

		err := UnprocessableEntity(map[string][]string{})
		WriteError(r.Context(), w, r, err)

		var env Envelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
		require.NotNil(t, env.Error)
		assert.Nil(t, env.Error.Detail)
	})
}

func TestWriteError_NonHTTPError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	plainErr := errors.New("something went wrong")
	WriteError(r.Context(), w, r, plainErr)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var env Envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "An internal error occurred", env.Error.Message)
	assert.NotEmpty(t, env.Error.TraceID)
	assert.Nil(t, env.Error.Detail)
	assert.Nil(t, env.Payload)
}

func TestWriteJSON(t *testing.T) {
	t.Run("with payload", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := map[string]string{"key": "value"}

		WriteJSON(w, http.StatusOK, payload)

		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		assert.Equal(t, "value", result["key"])
	})

	t.Run("nil payload", func(t *testing.T) {
		w := httptest.NewRecorder()

		WriteJSON(w, http.StatusNoContent, nil)

		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.Bytes())
	})
}
