## Компонент для создания базового приложения

`pkg/application` - это пакет для управления жизненным циклом приложения. Он отвечает за инициализацию, добавление компонентов (Kafka, Redis, Grpc и т.п.), запуск приложения и остановку в режиме gracefull shutdown.


### Пакет application состоит из следующих компонентов:

* Application - основная структура, которая представляет приложение. Она содержит поля для управления жизненным циклом приложения, такие как контекст, компоненты и middlewares.
* New - функция для создания нового экземпляра структуры Application. Она принимает контекст и массив опций и возвращает указатель на только что созданную структуру Application.
* Run - функция для запуска приложения.

### Компоненты (опции создания)

Пакет application включает в себя несколько компонентов, отвечающих за управление разными аспектами жизненного цикла приложения:

* WithOpenAPI - REST API из OpenAPI спецификаций (валидация, Swagger UI, генерация кода, автоматический запуск HTTP-сервера)
* WithKafka - компонент для добавления клиента Kafka. После создания приложения клиент доступен по адресу `app.Kafka`
* WithRedis - компонент для добавления клиента Redis. После создания приложения клиент доступен по адресу `app.Redis`
* WithTrace - компонент для запуска трассировки
* WithS3 - компонент для добавления клиента S3. После создания приложения клиент доступен по адресу `app.S3`
* WithWorkflow - компонент для подключения сервиса к оркестратору бизнес-процессов
* WithDb - компонент для подключения к БД Postgres, доступен через интерфейс `db.DbClient`
* WithLocalize - компонент добавления локализации

### Middlewares

Все middleware применяются автоматически к HTTP-серверу в следующем порядке:

1. **Panic Recovery** — перехватывает панику, возвращает JSON `{"error":"internal server error","correlation_id":"..."}` с HTTP 500. Детали паники логируются с correlation_id для поиска в логах.
2. **HTTP Metrics** — Prometheus метрики: `http_requests_total`, `http_request_duration_seconds` с лейблами method/path/status_code.
3. **Liveness** `/healthz/live` — возвращает 200 если httpServer жив.
4. **Readiness** `/healthz/ready` — проверяет что приложение перешло в состояние `started`.

При использовании `WithOpenAPI` дополнительно подключаются:

5. **Swagger UI** — `/swagger/` отдает интерактивную документацию, `/swagger/{name}/openapi.json` — спецификацию. При нескольких спецификациях отображается выпадающий список для выбора.
6. **Auth** — проверка аутентификации (по умолчанию заглушка с warn-логом).
7. **Rate Limiter** — ограничение количества запросов (по умолчанию заглушка с warn-логом).
8. **Validation** — автоматическая валидация запросов по всем OpenAPI-схемам (kin-openapi). При наличии нескольких спецификаций middleware последовательно ищет маршрут в каждой из них.

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

Описаны в [документе](../config/doc.md)

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

### REST API (WithOpenAPI)

Способ создания REST API сервиса. API описывается в одном или нескольких OpenAPI 3.x YAML файлах.
Весь Go-код генерируется одной командой `make generate`, разработчик реализует только бизнес-логику.

`WithOpenAPI` автоматически запускает HTTP-сервер.

#### Быстрый старт: создание нового сервиса

Разработчик создаёт только два файла, всё остальное генерируется автоматически.

**Шаг 1.** Создайте спецификацию. Пример: `api/openapi/items.yaml`:

```yaml
openapi: "3.0.0"
info:
  title: Items API
  version: "1.0.0"
paths:
  /items/{id}:
    get:
      operationId: getItemById
      parameters:
        - in: path
          name: id
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Item"
components:
  schemas:
    Item:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
```

**Шаг 2.** Создайте `cmd/app/main.go`:

```go
func main() {
    ctx := context.Background()
    app, err := application.New(ctx,
        application.WithOpenAPI(openapi.FS(), handler.Register),
    )
    if err != nil {
        panic(err)
    }
    app.Run()
}
```

**Шаг 3.** Зафиксируйте зависимость в 'internal/tools/tools.go':
```go
//go:build tools

package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
)
```

**Шаг 4.** Добавьте `Makefile`:

```makefile
PLATFORM_PATH  = $(shell go list -m -f '{{.Dir}}' easybnk.gitlab.yandexcloud.net/backend/platform-core)
SERVICE_ROOT   = $(shell pwd)

.PHONY: generate generate-check

generate:
	bash $(PLATFORM_PATH)/scripts/openapi-generate.sh $(SERVICE_ROOT)/api/openapi $(SERVICE_ROOT)

generate-check:
	CHECK=1 bash $(PLATFORM_PATH)/scripts/openapi-generate.sh $(SERVICE_ROOT)/api/openapi $(SERVICE_ROOT)
```

**Шаг 5.** Запустите генерацию:

```bash
make generate
```

Скрипт создаёт всё необходимое:

```
myservice/
├── api/
│   └── openapi/
│       ├── items.yaml                           # ВЫ СОЗДАЛИ
│       └── embed.go                             # СГЕНЕРИРОВАНО: go:embed спецификаций
├── cmd/app/
│   └── main.go                                  # ВЫ СОЗДАЛИ
├── internal/
│   ├── api/items/
│   │   └── api.gen.go                           # СГЕНЕРИРОВАНО: типы, интерфейсы, роутер
│   └── handler/
│       ├── register.go                          # СГЕНЕРИРОВАНО: агрегатор всех хэндлеров
│       └── items/
│           ├── handler.go                       # СГЕНЕРИРОВАНО: ItemsHandler с заглушками
│           └── register.go                      # СГЕНЕРИРОВАНО: регистрация на mux
└── Makefile
```

Имя структуры хэндлера (`ItemsHandler`) извлекается из `info.title` спецификации:
- `"Items API"` → `ItemsHandler`
- `"User Management API"` → `UserManagementHandler`

**Шаг 6.** Реализуйте бизнес-логику — замените `panic("not implemented")` в `internal/handler/items/handler.go`:

```go
func (h *ItemsHandler) GetItemById(ctx context.Context, req gen.GetItemByIdRequestObject) (gen.GetItemByIdResponseObject, error) {
    return gen.GetItemById200JSONResponse{Id: &req.Id, Name: ptr("Widget")}, nil
}
```

#### Повторная генерация

При повторном запуске `make generate`:
- `api/embed.go` — перегенерируется всегда
- `internal/api/*/api.gen.go` — перегенерируется всегда
- `internal/handler/register.go` — перегенерируется всегда (агрегатор, подхватывает новые спецификации)
- `internal/handler/*/handler.go` — **НЕ перезаписывается**, чтобы не потерять бизнес-логику
- `internal/handler/*/register.go` — **НЕ перезаписывается**

Если в спецификации появились новые операции, скрипт выведет предупреждение:
```
WARNING: handler.go is missing methods: DeleteItem
Add them manually or delete the file and re-run the scaffold.
```

#### Несколько спецификаций

Для нескольких доменов создайте отдельные YAML-файлы в `api/`:

```
api/openapi
├── items.yaml     # Items API
└── users.yaml     # Users API
```

`make generate` создаст отдельные пакеты для каждого:
```
internal/
├── api/
│   ├── items/api.gen.go
│   └── users/api.gen.go
└── handler/
    ├── register.go          # автоматически вызывает items.Register + users.Register
    ├── items/handler.go
    └── users/handler.go
```

Swagger UI на `/swagger/` покажет выпадающий список для выбора спецификации.

#### Кастомизация register.go

Сгенерированный `register.go` внутри `internal/handler/{name}/` можно доработать:

```go
// internal/handler/items/register.go
package items

func Register(ctx context.Context, mux *http.ServeMux) {
    db := di.Resolve[db.DbClient](ctx)
    repo := repository.New(db)
    h := &ItemsHandler{repo: repo}
    gen.HandlerFromMux(gen.NewStrictHandler(h, nil), mux)
}
```

При повторной генерации этот файл не будет перезаписан.

#### Конфигурация генератора

Конфигурация `oapi-codegen` хранится централизованно в platform-core (`pkg/openapi/codegen/cfg.yaml`).
Сервисы не содержат собственных конфигурационных файлов генератора — это обеспечивает
единообразие настроек генерации и исключает расхождения между сервисами.

#### Проверка в CI/CD

```bash
make generate-check
```

Скрипт запускает генерацию и проверяет, что результат совпадает с коммитом.

#### Автоматически доступно при запуске

- REST-эндпоинты на порту приложения
- Swagger UI на `/swagger/` (с выбором спецификации при наличии нескольких)
- Валидация запросов по всем OpenAPI-схемам
- Panic recovery, метрики, трассировка

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
	import "easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/config"

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
