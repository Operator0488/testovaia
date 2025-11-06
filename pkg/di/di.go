package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type contextKey struct{}

var (
	ErrAlreadyRegistered    = errors.New("already registered")
	ErrUnregisteredType     = errors.New("type is not registered")
	ErrNoContainer          = errors.New("container not found")
	ErrServiceMustBePointer = errors.New("instance must be a pointer")
)

var (
	globalContainer Container
	globalMu        sync.RWMutex
)

// SetGlobal устанавливает глобальный контейнер
func SetGlobal(container Container) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalContainer = container
}

func getGlobal() Container {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalContainer
}

// WithContainer добавляет контейнер в контекст, использовать для тестов
func WithContainer(ctx context.Context, container Container) context.Context {
	return context.WithValue(ctx, contextKey{}, container)
}

func fromContext(ctx context.Context) Container {
	if ctx == nil {
		return nil
	}
	if container, ok := ctx.Value(contextKey{}).(Container); ok {
		return container
	}
	return nil
}

func getContainer(ctx context.Context) Container {
	if ctx != nil {
		if container := fromContext(ctx); container != nil {
			return container
		}
	}

	if global := getGlobal(); global != nil {
		return global
	}

	panic(ErrNoContainer)
}

// Register регистрирует инстанс сервиса
// в T указывается интерфейс
// В instance указатель на структуру релизующую этот интерфейс
func Register[T any](ctx context.Context, instance T) T {
	container := getContainer(ctx)
	c := container.(*containerImpl)

	if err := c.register(reflect.TypeFor[T](), instance); err != nil {
		panic(err)
	}

	return instance
}

// RegisterFactory регистрирует фабричную функцию для создания сервиса
// factory должен вернуть указатель
func RegisterFactory[T any](ctx context.Context, factory func() T) {
	container := getContainer(ctx)
	c := container.(*containerImpl)

	instance := factory()

	if err := c.register(reflect.TypeFor[T](), instance); err != nil {
		panic(err)
	}
}

// Resolve возвращает инстанс сервиса
// можно вызывать до Build, но в таком случае будут возвращаться только те сервисы
// которые были уже зарегистрированы ранее
func Resolve[T any](ctx context.Context) T {
	container := getContainer(ctx)
	c := container.(*containerImpl)

	inst, err := c.resolve(reflect.TypeFor[T]())
	if err != nil {
		panic(fmt.Errorf("failed to resolve instance %w", err))
	}
	i, ok := inst.(T)
	if !ok {
		panic(fmt.Errorf("failed to cast instance %T", inst))
	}
	return i
}
