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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/response"
	pkgauth "easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/auth"
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
	routerList, err := buildRouters(context.Background(), [][]byte{minimalSpec})
	require.NoError(t, err)
	mw := httpValidationMiddleware(context.Background(), routerList)

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
	routerList, err := buildRouters(context.Background(), [][]byte{minimalSpec})
	require.NoError(t, err)
	mw := httpValidationMiddleware(context.Background(), routerList)

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
	routerList, err := buildRouters(context.Background(), [][]byte{minimalSpec})
	require.NoError(t, err)
	mw := httpValidationMiddleware(context.Background(), routerList)

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
	_, err := buildRouters(context.Background(), [][]byte{invalidSpec})
	assert.Error(t, err)
}

func TestNewMiddleware_BodyRestoredForHandler(t *testing.T) {
	routerList, err := buildRouters(context.Background(), [][]byte{minimalSpec})
	require.NoError(t, err)
	mw := httpValidationMiddleware(context.Background(), routerList)

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

// specWithSecurity — спека с doc-level security:[] и одним защищённым эндпоинтом
var specWithSecurity = []byte(`
openapi: "3.0.0"
info:
  title: Security Test API
  version: "1.0.0"
security: []
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
paths:
  /public:
    get:
      summary: Публичный
      responses:
        "200":
          description: OK
  /protected:
    post:
      summary: Защищённый
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: OK
`)

func TestAuthMiddleware_OpenAPISecurityRouting(t *testing.T) {
	routerList, err := buildRouters(context.Background(), [][]byte{specWithSecurity})
	require.NoError(t, err)

	authCalled := false
	app := newTestApp()
	app.openAPIRouters = routerList
	app.auth.fn = func(r *http.Request) (*http.Request, error) {
		authCalled = true
		return r, nil
	}

	t.Run("публичный эндпоинт пропускается без auth", func(t *testing.T) {
		authCalled = false
		req := httptest.NewRequest(http.MethodGet, "/public", nil)
		rec := httptest.NewRecorder()
		app.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})(rec, req)
		assert.False(t, authCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("защищённый эндпоинт вызывает auth", func(t *testing.T) {
		authCalled = false
		req := httptest.NewRequest(http.MethodPost, "/protected", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})(rec, req)
		assert.True(t, authCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("неизвестный маршрут считается защищённым", func(t *testing.T) {
		authCalled = false
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		rec := httptest.NewRecorder()
		app.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})(rec, req)
		assert.True(t, authCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// specWithScopes — спека для тестирования scope-ограничений
var specWithScopes = []byte(`
openapi: "3.0.0"
info:
  title: Scope Test API
  version: "1.0.0"
security: []
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
paths:
  /public:
    get:
      summary: Публичный
      security: []
      responses:
        "200":
          description: OK
  /any-token:
    get:
      summary: Любой валидный токен, scope не важен
      security:
        - BearerAuth: []
      responses:
        "200":
          description: OK
  /internal-only:
    get:
      summary: Только internal
      security:
        - BearerAuth: ["internal"]
      responses:
        "200":
          description: OK
  /or-scopes:
    get:
      summary: Internal ИЛИ external (два отдельных entry)
      security:
        - BearerAuth: ["internal"]
        - BearerAuth: ["external"]
      responses:
        "200":
          description: OK
  /and-scopes:
    get:
      summary: Internal И Supervisor одновременно (AND)
      security:
        - BearerAuth: ["internal", "supervisor"]
      responses:
        "200":
          description: OK
`)

func TestRouteSecurityRequirements(t *testing.T) {
	routerList, err := buildRouters(context.Background(), [][]byte{specWithScopes})
	require.NoError(t, err)

	tests := []struct {
		method       string
		path         string
		wantRequired bool
		wantScopes   [][]string // внешний слайс = OR-требования, внутренний = AND-scopes
	}{
		{http.MethodGet, "/public", false, nil},
		{http.MethodGet, "/any-token", true, [][]string{{}}},
		{http.MethodGet, "/internal-only", true, [][]string{{"internal"}}},
		{http.MethodGet, "/or-scopes", true, [][]string{{"internal"}, {"external"}}},
		{http.MethodGet, "/and-scopes", true, [][]string{{"internal", "supervisor"}}},
		{http.MethodGet, "/unknown", true, nil}, // fail-secure
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		reqs, required := routeSecurityRequirements(req, routerList)

		assert.Equal(t, tc.wantRequired, required, "path=%s required", tc.path)

		if tc.wantScopes == nil {
			assert.Empty(t, reqs, "path=%s scopes должны быть пусты", tc.path)
		} else {
			require.Len(t, reqs, len(tc.wantScopes), "path=%s кол-во OR-требований", tc.path)
			for i, wantGroup := range tc.wantScopes {
				for _, scheme := range reqs[i] {
					assert.ElementsMatch(t, wantGroup, scheme, "path=%s требование[%d]", tc.path, i)
				}
			}
		}
	}
}

func TestScopesSatisfied(t *testing.T) {
	makeReqs := func(groups ...[]string) openapi3.SecurityRequirements {
		reqs := make(openapi3.SecurityRequirements, len(groups))
		for i, g := range groups {
			reqs[i] = openapi3.SecurityRequirement{"BearerAuth": g}
		}
		return reqs
	}

	tests := []struct {
		tokenScopes []pkgauth.Scope
		reqs        openapi3.SecurityRequirements
		want        bool
	}{
		// пустой список scope = любой валидный токен
		{[]pkgauth.Scope{pkgauth.ScopeInternal}, makeReqs([]string{}), true},
		{[]pkgauth.Scope{pkgauth.ScopeExternal}, makeReqs([]string{}), true},

		// одиночный scope — точное совпадение
		{[]pkgauth.Scope{pkgauth.ScopeInternal}, makeReqs([]string{"internal"}), true},
		{[]pkgauth.Scope{pkgauth.ScopeExternal}, makeReqs([]string{"internal"}), false},
		{[]pkgauth.Scope{pkgauth.ScopeExternal}, makeReqs([]string{"external"}), true},
		{[]pkgauth.Scope{pkgauth.ScopeInternal}, makeReqs([]string{"external"}), false},

		// OR через два entry
		{[]pkgauth.Scope{pkgauth.ScopeInternal}, makeReqs([]string{"internal"}, []string{"external"}), true},
		{[]pkgauth.Scope{pkgauth.ScopeExternal}, makeReqs([]string{"internal"}, []string{"external"}), true},

		// AND: оба scope одновременно — токен с одним scope не проходит
		{[]pkgauth.Scope{pkgauth.ScopeInternal}, makeReqs([]string{"internal", "supervisor"}), false},
		{[]pkgauth.Scope{pkgauth.ScopeExternal}, makeReqs([]string{"internal", "supervisor"}), false},

		// AND: токен с несколькими scopes — проходит если есть все требуемые
		{[]pkgauth.Scope{pkgauth.ScopeInternal, "supervisor"}, makeReqs([]string{"internal", "supervisor"}), true},
		{[]pkgauth.Scope{pkgauth.ScopeInternal, "supervisor"}, makeReqs([]string{"internal"}), true},
		{[]pkgauth.Scope{pkgauth.ScopeInternal, "supervisor"}, makeReqs([]string{"external"}), false},
	}

	for _, tc := range tests {
		got := scopesSatisfied(tc.tokenScopes, tc.reqs)
		assert.Equal(t, tc.want, got,
			"tokenScopes=%q reqs=%v", tc.tokenScopes, tc.reqs,
		)
	}
}

func TestAuthMiddleware_ScopeEnforcement(t *testing.T) {
	routerList, err := buildRouters(context.Background(), [][]byte{specWithScopes})
	require.NoError(t, err)

	tests := []struct {
		path       string
		tokenScope pkgauth.Scope // пустая строка = нет токена (auth fn вернёт ошибку)
		wantCode   int
	}{
		{"/internal-only", pkgauth.ScopeInternal, http.StatusOK},
		{"/internal-only", pkgauth.ScopeExternal, http.StatusForbidden},
		{"/or-scopes", pkgauth.ScopeInternal, http.StatusOK},
		{"/or-scopes", pkgauth.ScopeExternal, http.StatusOK},
		{"/and-scopes", pkgauth.ScopeInternal, http.StatusForbidden},
		{"/and-scopes", pkgauth.ScopeExternal, http.StatusForbidden},
		{"/any-token", pkgauth.ScopeInternal, http.StatusOK},
		{"/any-token", pkgauth.ScopeExternal, http.StatusOK},
		{"/internal-only", "", http.StatusUnauthorized}, // нет токена
	}

	for _, tc := range tests {
		app := newTestApp()
		app.openAPIRouters = routerList
		app.auth.fn = func(r *http.Request) (*http.Request, error) {
			if tc.tokenScope == "" {
				return nil, errors.New("нет токена")
			}
			ctx := pkgauth.WithClaims(r.Context(), &pkgauth.Claims{Scopes: []pkgauth.Scope{tc.tokenScope}})
			return r.WithContext(ctx), nil
		}

		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		app.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})(rec, req)

		assert.Equal(t, tc.wantCode, rec.Code,
			"path=%s tokenScope=%q", tc.path, tc.tokenScope,
		)
	}
}
