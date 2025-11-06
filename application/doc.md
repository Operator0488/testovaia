## Компонент для создания базового приложения

`pkg/application` - это пакет для управления жизненным циклом приложения. Он отвечает за инициализацию, добавление компонентов (Kafka, Redis, Grpc и т.п.), запуск приложения и остановку в режиме gracefull shutdown.


### Пакет application состоит из следующих компонентов:

* Application - основная структура, которая представляет приложение. Она содержит поля для управления жизненным циклом приложения, такие как контекст, компоненты и middlewares.
* New - функция для создания нового экземпляра структуры Application. Она принимает контекст и массив опций и возвращает указатель на только что созданную структуру Application.
* Run - функция для запуска приложения.

### Компоненты (опции создания)

Пакет application включает в себя несколько компонентов, отвечающих за управление разными аспектами жизненного цикла приложения:

* WithKafka - компонент для добавления клиента Kafka. После создания приложения клиент доступен по адресу `app.Kafka`
* WithHTTP - компонент для запуска HTTP-сервера.
* WithRedis - компонент для добавления клиента Redis. После создания приложения клиент доступен по адресу `app.Redis`
* WithTrace - компонент для запуска трассировки.
* WithS3 - компонент для добавления клиента S3. После создания приложения клиент доступен по адресу `app.S3`
* WithWorkflow - компонент для подключения сервиса к оркестратору бизнес-процессов
* WithDb - компонент для подключения к БД Postgres, доступен через интерфейс `db.DbClient`
* WithLocalize - компонент добавления локализации

### Middlewares

На данный момент релизованы мидлвари для снятия k8s проб по HTTP-адресам, данные мидлвари автоматически применяются к поднятому http-серверу

##### Пробы для `/healthz/live`
Возвращает 200 если компонент httpServer жив
##### Пробы для  `/healthz/ready`
Проверяет что приложение перешло в состояние `started`. Приложение переходит в состояние `started` после успешной инициализации и запуска всех компонентов, которые были добавлены через `WithComponent`, а так же проверяет что health check всех компонентов не возвращают ошибки.

### Дополнительные методы

- `app.Env` - доступ к конфигурации
- `app.RegisterRouter(e)` - дает возможность зарегистрировать кастомный роутинг например `echo` для HTTP-методов приложения (см. раздел с примерами)
- `app.Closer.Add(someFunc)` - дает возможность зарегистрировать функцию, которая должна выполнится при gracefull shutdown (см. раздел с примерами)
- `app.Health.Add(name, healthFunc)` - дает возможность зарегистрировать функцию, которая будет выполнятся при проверке здоровья сервиса

### Доступные переменные в config.yaml

Описаны в [документе](../../docs/config.md)

### Использование di для получения зависимостей

Все компоненты которые были добавлены через `Option` при создании app, так же будут доступны через di контейнер.
Пример: 

```go

	// добавили компонент S3
	app, err := application.New(
		ctx,
		application.WithS3(),
	)

	// теперь мы можем использовать его через di
	s3Client:=di.Resolve[s3client.Client](ctx)

```

### Примеры использования http-сервер

```go
	ctx := context.Background()

	app, err := application.New(
		ctx,
		application.WithHTTP(),
	)
	if err != nil {
		logger.Fatal(ctx, "failed create app", logger.Err(err))
		return
	}

	// роутинг для HTTP-сервера
	e := echo.New()
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"test": "200"})
	})

	// регистрация кастомного роутеа
	app.RegisterRouter(e)

	// старт приложения
	app.Run()
```

### Примеры использования Kafka

```go
	ctx := context.Background()

	app, err := application.New(
		ctx,
		application.WithKafka(),
	)
	if err != nil {
		logger.Fatal(ctx, "failed create app", logger.Err(err))
		return
	}

	// создание продюсеров
	p1, err := app.Kafka.RegisterProducer(ctx, "topic_name", kafka.WithTopicCreate(kafka.CreateTopicConfig{NumPartitions: 5}))
	p2, err := app.Kafka.RegisterProducer(ctx, "topic_name2", kafka.WithTopicCreate(kafka.CreateTopicConfig{NumPartitions: 5}))

	// создание консуюмеров
	app.Kafka.RegisterConsumer(ctx, "topic_name", func(ctx context.Context, msg kafka.Message) error {
		// обработка сообщения
		return nil
	})

	// старт приложения
	app.Run()
```

### Пример использования S3

```go
	ctx := context.Background()

	app, err := application.New(
		ctx,
		application.WithS3(),
	)
	if err != nil {
		logger.Fatal(ctx, "failed create app", logger.Err(err))
		return
	}

	// использование инициализированного клиента S3
	appUsecase:=usecase.New(app.S3)
	
	// старт приложения
	app.Run()
```

### Пример использования Workflow

```go
	ctx := context.Background()

	app, err := application.New(
		ctx,
		application.WithWorkflow(),
	)
	if err != nil {
		logger.Fatal(ctx, "failed create app", logger.Err(err))
		return
	}

	app.Workflow.
	    // Декларация бизнес-процесса, которым владеет сервис
		// Сервис может не иметь собственных процессов
		WithProcess("bpmn/myServiceProcess.bpmn").
	    // Подписка на задачу по типу с дефолтной конфигурацией 
		WithHandler(constants.SomeTaskName, func(ctx context.Context, task WorkflowTask) error {
		    // Обработка задачи
			return nil	
        }).
		// Подписка на задачу с кастомной конфигурацией
		WithHandler(
			constants.SomeTaskName2,
            func(ctx context.Context, task WorkflowTask) error {
			    // Обработка задачи
				return nil	
            },
            TaskHandlerConfig {
                IncidentMaxRetries: 1,  // Количество ретраев до инцидента
				MaxActiveTasks: 3,      // Количество параллельно выполняемых задач этого типа
            })

	// старт приложения
	app.Run()
```

### Пример использования Postgres

```go

	// main.go
	ctx := context.Background()

	app, err := application.New(
		ctx,
		application.WithDB(),
	)
	if err != nil {
		logger.Fatal(ctx, "failed create app", logger.Err(err))
		return
	}


	// старт приложения
	app.Run()


	// repository.go

	type userRepo struct {
		Client
	}

	// Использование DB(ctx)
	func (u *userRepo) GetUser(ctx context.Context, id string) (*domain.User, error) {
		var user domain.User
		err := u.DB(ctx).First(&user, id).Error
		return &user, err
	}

	func (u *userRepo) CreateUser(ctx context.Context, user *domain.User) error {
		return u.DB(ctx).Create(user).Error
	}

	// Получение зависимости через di
	func (u *userRepo) ResolveDeps(client Client) {
		u.Client = client
	}

```

### Пример использования Postgres в транзакции



```go

	// user_repo.go

	type userRepo struct {
		Client
	}

	func (u *userRepo) CreateUser(ctx context.Context, user *domain.User) error {
		return u.DB(ctx).Create(user).Error
	}

	// Получение зависимости через di
	func (u *userRepo) ResolveDeps(client Client) {
		u.Client = client
	}

	// role_repo.go

	type roleRepo struct {
		Client
	}

	func (u *userRepo) CreateRole(ctx context.Context, user *domain.Role) error {
		return u.DB(ctx).Create(user).Error
	}

	// Получение зависимости через di
	func (u *userRepo) ResolveDeps(client Client) {
		u.Client = client
	}

	// user_service.go

	type userService struct {
		Client
		users IUsersRepo
		roles IRolesRepo
	}

	// Транзакционное создание роли и пользователя при передаче контекста tctx
	func (u *userService) CreateUserWithRole(ctx context.Context, user *domain.Role, role *domain.Role) error {
		return u.WithTransaction(ctx, func(tctx context.Context) error { // при возврате ошибки транзакция будет отменена
			if err:=u.roles.CreateRole(tctx, role);err!=nil {
				return err
			}
			if err:=u.roles.CreateUser(tctx, user);err!=nil {
				return err 
			}
			return nil
		})
	}

	// Получение зависимостей через di
	func (u *userService) ResolveDeps(client Client, u IUsersRepo, r IRolesRepo) {
		u.Client = client
		u.users = u
		u.roles = r
	}

```

### Пример регистрации в gracefull shutdown кастомных компонентов

```go
	ctx := context.Background()

	app, err := application.New(
		ctx,
	)
	if err != nil {
		logger.Fatal(ctx, "failed create app", logger.Err(err))
		return
	}
	
	telegramBot := telegram.NewBot("token")

	// регистрация кастомных Closers, например если нужно выгрузить какие-то компоненты, которых нет в стандартной библиотеке
	app.Closer.Add(telegramBot.Close)

	app.Health.Add("telegram_bot", func(ctx context.Context) error {
		return bot.Alive(ctx)
	})

	// старт приложения
	app.Run()
```

### Порядок запуска мидлвари

```go
 // если есть recovery то его первым
a.middlewares.Add(a.httpMetricsMiddleware)
a.middlewares.Add(a.livenessMiddleware)
a.middlewares.Add(a.readinessMiddleware)
```

### Пример регистрации в Healthcheck кастомных компонентов

```go
	ctx := context.Background()

	app, err := application.New(
		ctx,
	)
	if err != nil {
		logger.Fatal(ctx, "failed create app", logger.Err(err))
		return
	}
	
	telegramBot := telegram.NewBot("token")
	
	// регистрация кастомной функции health check для кастомного компонента
	app.Health.Add("telegram_bot", func(ctx context.Context) error {
		return bot.Alive(ctx)
	})

	// старт приложения
	app.Run()
```

### Пример регистрации Zero Down Time конфигурации

Обновление конфигурации происходит путем создания обертки над конфигом через конструкторв `config.NewConfigWatcher`. Затем нужно подписаться на обновления через `app.Env.Subscribe(watcher)`.

```go

	p1:="some_name" // строковое название, исключительно для удобства логирования
	
	p2:=app.Env // конфиг приложения

	// функция создания конфига компонента
	p3:=func(c config.Configurer) someComponentConfig { 
		return someComponentConfig{
			RateLimitMax: c.GetInt("rate_limit.max_limit")
			// ... other fields
		}
	}

	watcher:=config.NewConfigWatcher(p1, p2, p3)
```

```go
	import "git.vepay.dev/knoknok/backend-platform/pkg/config"

	type usecaseConfig struct {
		RateLimitMax int
	}

	type usecase struct {
		config config.IConfigWatcher[usecaseConfig]
	}

	func NewUsecase(cfg config.Configurer) *usecase {
		// создание вотчера конфига
		configWatcher := config.NewConfigWatcher("app_config", cfg, func(c config.Configurer) usecaseConfig {
			return usecaseConfig{RateLimitMax: c.GetInt("rate_limit.max_limit")}
		})

		u := &usecase{config: configWatcher}

		configWatcher.OnRefresh(u.rebuildUsecase) // регистрация колбека, например если нужно переинициализировать компонент

		cfg.Subscribe(configWatcher) // подписка на обновления конфига
		return u
	}

	// чтение конфига через .Get() 
	func (u *usecase) GetRateLimit() int {
		return u.config.Get().RateLimitMax
	}

	// функция переинициализации компонента
	func (u *usecase) rebuildUsecase(cfg usecaseConfig) error {
		fmt.Println("reinitialize process", cfg)
		return nil
	}
```