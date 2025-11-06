package application

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

var (
	ErrHTTPServerNotFound = errors.New("http router not defined, use app.RegisterRouter")
	httpServer            = NewComponent("http", Noop, runHTTP)
)

// WithHTTP add httpServer component, started on HTTP_PORT.
func WithHTTP() Option {
	return func(app *Application) error {
		if ok := app.components.add(component(httpServer)); !ok {
			return ErrComponentAlreadyExist
		}
		return nil
	}
}

// RegisterRouter add custom router: echo, gin, etc.
func (a *Application) RegisterRouter(router http.Handler) {
	a.router = router
}

func runHTTP(ctx context.Context, a *Application) error {
	if a.router == nil {
		return ErrHTTPServerNotFound
	}

	a.httpServer = a.createServer()
	a.httpServer.Handler = a.middlewares.Chain()(a.router.ServeHTTP)

	a.Closer.Add(func() error {
		return a.httpServer.Shutdown(ctx)
	})

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		wg.Done()
		logger.Info(ctx, "Running HTTP server", logger.String("addr", a.httpServer.Addr))
		if a.testMode {
			return
		}
		err := a.httpServer.ListenAndServe()
		// если получили ошибку которая появилась не из-за шатдауна то надо убивать приложение
		if err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "Application component error",
				logger.String("component", "http"),
				logger.Err(err),
			)
			a.stop()
		}
	}()
	wg.Wait()
	return nil
}

func (a *Application) createServer() *http.Server {
	cfg := a.config.GetHTTPServerConfig()
	return &http.Server{
		Addr:         cfg.GetAddr(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}
