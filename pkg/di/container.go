package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Container интерфейс DI контейнера
type Container interface {
	// Build инициализирует фабрики и запускает процесс инжектирования
	Build() error
}

type dependency struct {
	service  any
	injected bool
}

type containerImpl struct {
	mu           sync.RWMutex
	dependencies map[string]dependency // сервисы по имени пакета и типа
	initialized  bool                  // флаг инициализации
}

// New создает новый экземпляр контейнера
func New() Container {
	return &containerImpl{
		dependencies: make(map[string]dependency),
	}
}

// Build инициализирует все зарегистрированные фабрики
func (c *containerImpl) Build() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Если уже инициализирован, ничего не делаем
	if c.initialized {
		return nil
	}

	if err := c.build(); err != nil {
		return err
	}

	c.initialized = true
	return nil
}

func (c *containerImpl) exist(typeName string) error {
	if _, exists := c.dependencies[typeName]; exists {
		return fmt.Errorf("%w in services for type_name: %s", ErrAlreadyRegistered, typeName)
	}
	return nil
}

func (c *containerImpl) build() error {
	var err error
	for _, dep := range c.dependencies {
		if dep.injected {
			continue
		}
		ierr := c.inject(dep.service)
		err = errors.Join(err, ierr)
	}

	return err
}

func (c *containerImpl) getDependency(paramType reflect.Type) (dependency, error) {
	typeName := getTypeName(paramType)

	dep, exists := c.dependencies[typeName]
	if !exists {
		return dependency{}, fmt.Errorf("%w: %s", ErrUnregisteredType, paramType.Name())
	}
	return dep, nil
}

func (c *containerImpl) resolve(paramType reflect.Type) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dep, err := c.getDependency(paramType)
	if err != nil {
		return nil, err
	}

	return dep.service, nil
}

func (c *containerImpl) register(typeInfo reflect.Type, instance any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	typeName := getTypeName(typeInfo)
	if err := c.exist(typeName); err != nil {
		panic(err)
	}

	if c.initialized {
		if err := c.inject(instance); err != nil {
			return err
		}
		c.dependencies[typeName] = dependency{service: instance, injected: true}
		return nil
	}

	// пробуем инжектить параметры
	err := c.inject(instance)
	if err == nil {
		c.dependencies[typeName] = dependency{service: instance, injected: true}
		return nil
	}

	// если не удалось собрать все зависимости, то ждем Build
	if errors.Is(err, ErrUnregisteredType) {
		c.dependencies[typeName] = dependency{service: instance, injected: false}
		return nil
	}

	return err
}

// inject находит метод ResolveDeps у переданного инстанса и внедряет зависимости
func (c *containerImpl) inject(instance interface{}) error {
	val := reflect.ValueOf(instance)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("%w, got %T", ErrServiceMustBePointer, instance)
	}

	method := val.MethodByName("ResolveDeps")
	if !method.IsValid() {
		return nil
	}

	methodType := method.Type()
	numIn := methodType.NumIn()

	args := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		paramType := methodType.In(i)

		dep, err := c.getDependency(paramType)
		if err != nil {
			return fmt.Errorf("failed to resolve dependency for %s, param %d, %w", reflect.TypeOf(instance), i, err)
		}

		args[i] = reflect.ValueOf(dep.service)
	}

	results := method.Call(args)

	// Обрабатываем возвращаемое значение (предполагаем error)
	if len(results) > 0 {
		if errVal := results[0]; errVal.IsValid() && !errVal.IsNil() {
			if err, ok := errVal.Interface().(error); ok {
				return err
			}
		}
	}

	return nil
}

// getTypeName возвращает имя типа для использования в качестве ключа
func getTypeName(typeOf reflect.Type) string {
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}

	// Возвращаем полное имя с путем пакета
	return typeOf.PkgPath() + "." + typeOf.Name()
}
