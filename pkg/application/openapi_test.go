package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/config"
)

var openAPISpec = []byte(`
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
`)

func specFS(specs ...[]byte) fstest.MapFS {
	m := fstest.MapFS{}
	if len(specs) == 1 {
		m["test.yaml"] = &fstest.MapFile{Data: specs[0]}
	} else {
		for i, s := range specs {
			name := string(rune('a'+i)) + ".yaml"
			m[name] = &fstest.MapFile{Data: s}
		}
	}

	return m
}

func TestWithOpenAPI_RegistersComponentAndRouter(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env")))

	called := false
	app, err := newApp(
		ctx,
		config.GetConfig(),
		WithOpenAPI(specFS(openAPISpec), func(_ context.Context, mux *http.ServeMux) {
			called = true
			mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			})
		}))
	require.NoError(t, err)

	assert.True(t, called, "RegisterFn must be called during init")
	assert.Contains(t, app.components.list, "openapi")
	assert.NotNil(t, app.apiFS)
	assert.NotNil(t, app.registerFn)
}

func TestWithOpenAPI_ValidRequestReachesHandler(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env")))

	app, err := newApp(
		ctx,
		config.GetConfig(),
		WithOpenAPI(specFS(openAPISpec), func(_ context.Context, mux *http.ServeMux) {
			mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			})
		}))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Alice"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.middlewares.Chain()(app.router.ServeHTTP)(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestWithOpenAPI_InvalidRequestRejectedByMiddleware(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env")))

	handlerCalled := false
	app, err := newApp(
		ctx,
		config.GetConfig(),
		WithOpenAPI(specFS(openAPISpec), func(_ context.Context, mux *http.ServeMux) {
			mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusCreated)
			})
		}))
	require.NoError(t, err)

	// missing required "name" field
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.middlewares.Chain()(app.router.ServeHTTP)(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, handlerCalled, "handler must not be called when validation fails")

	var resp struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "request validation failed")
}

func TestWithOpenAPI_DuplicateReturnsError(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env")))

	reg := func(_ context.Context, mux *http.ServeMux) {}
	_, err := newApp(ctx, config.GetConfig(),
		WithOpenAPI(specFS(openAPISpec), reg),
		WithOpenAPI(specFS(openAPISpec), reg),
	)

	assert.ErrorIs(t, err, ErrComponentAlreadyExist)
}

func TestWithOpenAPI_InvalidSpecFailsInit(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env")))

	_, err := newApp(
		ctx,
		config.GetConfig(),
		WithOpenAPI(specFS(invalidSpec), func(_ context.Context, mux *http.ServeMux) {}),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed init openapi component")
}

func TestWithOpenAPI_UnknownRoutePassesThrough(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env")))

	app, err := newApp(
		ctx,
		config.GetConfig(),
		WithOpenAPI(specFS(openAPISpec), func(_ context.Context, mux *http.ServeMux) {
			mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
		}))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	app.middlewares.Chain()(app.router.ServeHTTP)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWithOpenAPI_EmptyFSReturnsError(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env")))

	emptyFS := fstest.MapFS{}
	_, err := newApp(
		ctx,
		config.GetConfig(),
		WithOpenAPI(emptyFS, func(_ context.Context, mux *http.ServeMux) {}),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no YAML specs found")
}

func TestWithOpenAPI_EncapsulatesHTTP(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env")))

	app, err := newApp(
		ctx,
		config.GetConfig(),
		WithOpenAPI(specFS(openAPISpec), func(_ context.Context, mux *http.ServeMux) {
			mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			})
		}))
	require.NoError(t, err)

	assert.Contains(t, app.components.list, "http",
		"WithOpenAPI must auto-register the HTTP server component")
}
