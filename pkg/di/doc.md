## Пакет DI

Пакет `di` предоставляет простую систему внедрения зависимостей (Dependency Injection) для Go приложений.

Пакет позволяет регистрировать и разрешать зависимости через глобальный контейнер или контекст-специфичные контейнеры. Поддерживает регистрацию как готовых экземпляров, так и фабричных функций.

### Управление контейнером 

```go
// SetGlobal устанавливает глобальный контейнер
func SetGlobal(container Container)

// WithContainer добавляет контейнер в контекст (для тестирования)
func WithContainer(ctx context.Context, container Container) context.Context
```

### Регистрация зависимостей

```go
// Register регистрирует экземпляр сервиса
// T - интерфейс, instance - указатель на структуру, реализующую интерфейс
func Register[T any](ctx context.Context, instance T)

// RegisterFactory регистрирует фабричную функцию для создания сервиса
// factory должен вернуть указатель на экземпляр
func RegisterFactory[T any](ctx context.Context, factory func() T)
```

### Разрешение зависимостей

Чтобы получить зарегистрированную зависимость используется функция `Resolve`.

```go
// Resolve возвращает экземпляр сервиса
// Можно вызывать до Build(), но вернет только уже зарегистрированные сервисы
func Resolve[T any](ctx context.Context) T
```

Так можно использовать метод `ResolveDeps` чтобы получить все зависимости для своего сервиса.
Например есть сервис `superServiceRepository`, у которого есть зависимости в виде `pgdb.IUserRepository` и `clickdb.IUserRepository`
Чтобы они автоматически подтянулись при регистрации данного сервиса нужно объявить у этого сервиса `ResolveDeps` как в примере ниже, и они автоматически будут подтянуты после операции `Build`

```go
// super_service.go
type superServiceRepository struct {
	repoPg    pgdb.IUserRepository
	repoClick clickdb.IUserRepository
}

func (m *superServiceRepository) ResolveDeps(repo1 pgdb.IUserRepository, repo2 clickdb.IUserRepository) {
	m.repoPg = repo1
	m.repoClick = repo2
}

// main.go
di.Register[clickdb.IUserRepository](ctx, &clickdb.UserRepository{})
di.Register[pgdb.IUserRepository](ctx, &pgdb.UserRepository{})


// после регистраци будет вызван метод ResolveDeps который добавит нужные зависимости в этот сервис
 di.Register[ISuperServiceRepository](ctx, &superServiceRepository{}) 

```

### Пример использования

```go
// Определение интерфейса и реализации
type Service interface {
    DoSomething()
}

type ServiceImpl struct{}

func (s *ServiceImpl) DoSomething() {
    fmt.Println("Service working")
}

// Регистрация в main или init
func main() {
    container := NewContainer()
    di.SetGlobal(container)
    
    ctx := context.Background()
    
    // Регистрация сервиса
    di.Register[Service](ctx, &ServiceImpl{})
    
    // Или через фабрику
    di.RegisterFactory[Service](ctx, func() Service {
        return &ServiceImpl{}
    })
    
    // Разрешение зависимости
    service := di.Resolve[Service](ctx)
    service.DoSomething()
}
```

### Тестирование

Поддерживает тестирование через контекст-специфичные контейнеры

Пример
```go
func TestRegister(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer) // добавление контейнера в контекст

	Register[pgdb.IUserRepository](ctx, &pgdb.UserRepository{User: "test"})

	err := testContainer.Build()
	require.NoError(t, err)

	myService := Resolve[pgdb.IUserRepository](ctx)
	require.NotNil(t, myService)

	assert.Equal(t, "test", myService.GetProfile())
}
```