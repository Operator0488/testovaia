package application

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/grpc/server"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/swagger"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/workflow"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/closers"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/config"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/health"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/middleware"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/translations"
	cfg "easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/config"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/db"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/di"
	grpcclient "easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/grpc/client"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/kafka"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/localize"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/redis"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/s3client"
)

const (
	defaultWaitCloserTime = time.Second * 30
)

type Application struct {
	Name      string
	Closer    closers.Closer
	Health    health.Health
	Env       config.Configurer
	config    appConfig
	testMode  bool
	container di.Container
	workflow  workflow.Workflow

	// private

	started        chan struct{} // флаг-сигнал приложение успешно стартовало
	closed         chan struct{} // флаг сигнал процесс gracefull shutdown завершен
	closing        chan struct{} // флаг-сигнал начался процесс gracefull shutdown
	shutdown       chan os.Signal
	context        context.Context
	components     *components
	middlewares    middleware.Middlewares
	waitCloserTime time.Duration // wait closers time

	// initializing components
	Redis            redis.Redis
	DB               db.DbClient
	Kafka            kafka.KafkaClient
	Workflow         workflow.WorkflowBuilder
	S3               s3client.Client
	router           http.Handler
	httpServer       *http.Server
	Localizer        localize.Localizer
	translateManager translations.TranslateManager

	// GRPC
	PrivateGrpcServer *grpcserver.Manager
	PublicGrpcServer  *grpcserver.Manager
	GrpcClients       *grpcclient.Manager

	// HTTP / OpenAPI
	swagger    *swagger.Manager
	apiFS      fs.FS
	registerFn RegisterFn

	auth      *authConfig
	rateLimit *rateLimitConfig
}

func NewWithConfig(ctx context.Context, env config.Configurer, components ...Option) (*Application, error) {
	return new(ctx, env, components...)
}

func New(ctx context.Context, components ...Option) (*Application, error) {
	logger.Info(ctx, "Application creating")
	err := cfg.Init(ctx)
	if err != nil {
		logger.Error(ctx, "Application config failed", logger.Err(err))
		return nil, err
	}

	logger.Info(ctx, "Application config initilized")

	env := cfg.GetConfig()

	return new(ctx, env, components...)
}

func new(ctx context.Context, env config.Configurer, components ...Option) (*Application, error) {
	app := &Application{
		Closer:         closers.New(),
		Health:         health.New(),
		middlewares:    middleware.New(),
		components:     newComponents(),
		started:        make(chan struct{}, 1),
		closed:         make(chan struct{}, 1),
		closing:        make(chan struct{}, 1),
		shutdown:       make(chan os.Signal, 1),
		Env:            env,
		config:         appConfig{env},
		context:        ctx,
		router:         http.HandlerFunc(noopHandler()),
		waitCloserTime: defaultWaitCloserTime,
		auth:           &authConfig{},
		rateLimit:      &rateLimitConfig{},
	}

	// добавляем компонент контейнера первым,
	// на этапе init создается контейнер
	// на этапе run запускается создание фабрик и инжектирование зависимостей
	app.components.add(component(containerComponent))

	for _, c := range components {
		if err := c(app); err != nil {
			return nil, err
		}
	}

	app.middlewares.Add(app.panicRecoveryMiddleware)
	app.middlewares.Add(app.httpMetricsMiddleware)

	// добавление k8s мидлваров
	app.addProbes()

	// подписка на изменения и регистрация в di
	app.initConfig(ctx)

	logger.Info(ctx, "Application components initializing")
	if err := app.components.init(ctx, app); err != nil {
		logger.Error(ctx, "Application components failed", logger.Err(err))
		return nil, err
	}

	logger.Info(ctx, "Application created")

	signal.Notify(app.shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go app.finish()

	return app, nil
}

func (a *Application) finish() {
	signal := <-a.shutdown
	logger.Info(a.context, "Application shutdown started", logger.String("signal", signal.String()))

	close(a.closing)
	ctx, cancel := context.WithTimeout(a.context, a.waitCloserTime)
	defer cancel()
	if err := a.Closer.Close(ctx); err != nil {
		logger.Error(a.context, "Application finish error", logger.Err(err))
	}

	logger.Info(a.context, "Application shutdown finished")
	close(a.closed)
}

func (a *Application) stop() {
	a.shutdown <- os.Interrupt
}

func (a *Application) Run() error {
	ctx, cancel := context.WithCancel(a.context)
	defer cancel()

	if err := a.components.run(ctx, a); err != nil {
		logger.Error(ctx, "Application finished with error", logger.Err(err))
		return err
	}

	logger.Info(a.context, "Application running")
	close(a.started)

	<-a.closed

	logger.Info(a.context, "Application exit")
	return nil
}
