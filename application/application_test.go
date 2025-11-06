package application

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	cfg "git.vepay.dev/knoknok/backend-platform/internal/pkg/config"
	"git.vepay.dev/knoknok/backend-platform/internal/pkg/mock"
	"git.vepay.dev/knoknok/backend-platform/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	os.Setenv(config.EnvConsulDisabled, "true")
	os.Setenv(config.EnvVaultDisabled, "true")

	code := m.Run()

	os.Unsetenv(config.EnvConsulDisabled)
	os.Unsetenv(config.EnvVaultDisabled)

	os.Exit(code)
}

func newApp(ctx context.Context, env cfg.Configurer, components ...Option) (*Application, error) {
	app, err := NewWithConfig(ctx, config.GetConfig(), components...)
	if err != nil {
		return nil, err
	}
	app.testMode = true
	return app, nil
}

func TestNewWithConfig(t *testing.T) {
	ctx := context.Background()
	err := config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env"))
	assert.NoError(t, err)
	app, err := newApp(ctx, config.GetConfig())
	assert.NoError(t, err)
	assert.NotNil(t, app)
}

func TestWithComponent(t *testing.T) {
	ctx := context.Background()
	err := config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env"))
	assert.NoError(t, err)
	app, err := newApp(ctx, config.GetConfig(), WithComponent(
		"test",
		func(ctx context.Context, a *Application) error {
			return nil
		},
		func(ctx context.Context, a *Application) error {
			return nil
		},
	))
	assert.NoError(t, err)
	assert.Equal(t, 3, len(app.components.list)) // with probes and DI
	assert.NotNil(t, app.components.list["test"])
	assert.NotNil(t, app.components.list["http"])
}

func TestComponents(t *testing.T) {
	ctx := context.Background()
	err := config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env"))
	assert.NoError(t, err)
	app, err := newApp(ctx, config.GetConfig(), WithComponent(
		"test",
		func(ctx context.Context, a *Application) error {
			return nil
		},
		func(ctx context.Context, a *Application) error {
			return nil
		},
	))
	assert.NoError(t, err)
	assert.Equal(t, 3, len(app.components.list)) // with probes and DI container
	assert.NotNil(t, app.components.list["test"])
	assert.NotNil(t, app.components.list["http"])
}

func TestRunWithMocks(t *testing.T) {
	ctx := context.Background()
	err := config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env"))
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := mock.NewMockTestComponent(ctrl)

	mock.EXPECT().Init(gomock.Any()).Times(1)
	mock.EXPECT().Run(gomock.Any()).Times(1)

	app, err := newApp(ctx, config.GetConfig(), WithComponent(
		"test",
		func(ctx context.Context, a *Application) error {
			return mock.Init(ctx)
		},
		func(ctx context.Context, a *Application) error {
			return mock.Run(ctx)
		},
	))
	assert.NoError(t, err)
	waitRun(t, ctx, app)
}

func TestLivenessMiddleware(t *testing.T) {
	ctx := context.Background()
	err := config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env"))
	assert.NoError(t, err)
	app, err := newApp(ctx, config.GetConfig())
	assert.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   HealthResponse
		router         http.Handler
	}{
		{
			name:           "health endpoint",
			path:           "/healthz/live",
			expectedStatus: http.StatusOK,
			expectedBody: HealthResponse{
				Status:  "healthy",
				Message: "Application is running",
				Code:    http.StatusOK,
			},
		},
		{
			name:           "other endpoint passes through",
			path:           "/api/test",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "found endpoint",
			path:           "/api/account",
			expectedStatus: http.StatusOK,
			router: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.router != nil {
				app.RegisterRouter(tt.router)
			}

			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler := app.middlewares.Chain()(app.router.ServeHTTP)
			handler(rr, req)

			if tt.path == "/healthz/live" {
				assert.Equal(t, tt.expectedStatus, rr.Code)

				var response HealthResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.expectedBody.Status, response.Status)
				assert.Equal(t, tt.expectedBody.Code, response.Code)
			} else {
				assert.Equal(t, tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestCheckReadiness(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env"))
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	mock := mock.NewMockTestComponent(ctrl)

	app, err := newApp(ctx, config.GetConfig())

	app.Health.Add("test", mock.HealthCheck)

	assert.NoError(t, err)

	waitRun(t, ctx, app)

	tests := []struct {
		name   string
		status string
		mock   func()
	}{
		{
			name:   "not_ready",
			status: "not_ready",
			mock: func() {
				mock.EXPECT().HealthCheck(gomock.Any()).Return(errors.New("not yet ready"))
			},
		},
		{
			name:   "ready",
			status: "ready",
			mock: func() {
				mock.EXPECT().HealthCheck(gomock.Any()).Return(nil)
			},
		},
		{
			name:   "shutdown",
			status: "not_ready",
			mock: func() {
				app.stop()
				<-app.closing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			res := checkReadiness(ctx, app)
			assert.Equal(t, tt.status, res.Status, res)
		})
	}
}

func TestClosers(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env"))
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	mock := mock.NewMockTestComponent(ctrl)

	app, err := NewWithConfig(ctx, config.GetConfig())
	assert.NoError(t, err)

	app.Closer.Add(func() error {
		return mock.Close(context.TODO())
	})

	mock.EXPECT().Close(gomock.Any()).Return(nil)

	waitRun(t, ctx, app)

	app.stop()

	select {
	case <-app.closed:
	case <-ctx.Done():
		assert.FailNow(t, "test timeout")
	}
}

func TestClosersLong(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := config.Init(ctx, config.WithConfigPath("./"), config.WithFileName("test.env"))
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	mock := mock.NewMockTestComponent(ctrl)

	app, err := newApp(ctx, config.GetConfig())
	assert.NoError(t, err)

	app.Closer.Add(func() error {
		time.Sleep(time.Second * 10)
		return mock.Close(context.TODO())
	})

	mock.EXPECT().Close(gomock.Any()).Times(0)

	waitRun(t, ctx, app)

	app.waitCloserTime = time.Millisecond * 100
	app.stop()

	select {
	case <-app.closed:
	case <-ctx.Done():
		assert.FailNow(t, "test timeout")
	}
}

func waitRun(t *testing.T, ctx context.Context, app *Application) {
	go func() {
		app.Run()
	}()

	select {
	case <-app.started:
	case <-ctx.Done():
		assert.FailNow(t, "test timeout")
	}
}
