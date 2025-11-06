package application

import (
	"context"
	"fmt"
)

type ComponentFunc func(context.Context, *Application) error

type Component component

type component struct {
	name   string
	initFn ComponentFunc
	runFn  ComponentFunc
}

type components struct {
	order []string
	list  map[string]component
}

func Noop(context.Context, *Application) error {
	return nil
}

func newComponents() *components {
	return &components{
		list: make(map[string]component, 0),
	}
}

func (e *components) has(name string) bool {
	_, ok := e.list[name]
	return ok
}

func (e *components) add(ent component) bool {
	if e.has(ent.name) {
		return false
	}
	e.order = append(e.order, ent.name)
	e.list[ent.name] = ent
	return true
}

func (e *components) addFirst(ent component) bool {
	if e.has(ent.name) {
		return false
	}
	e.order = append([]string{ent.name}, e.order...)
	e.list[ent.name] = ent
	return true
}

func (e *components) init(ctx context.Context, a *Application) error {
	for _, key := range e.order {
		item := e.list[key]
		e := item.initFn(ctx, a)
		if e != nil {
			return fmt.Errorf("failed init %s component, error: %w", item.name, e)
		}
	}
	return nil
}

func (e *components) run(ctx context.Context, a *Application) error {
	for _, key := range e.order {
		item := e.list[key]
		e := item.runFn(ctx, a)
		if e != nil {
			return fmt.Errorf("failed run %s component, error: %w", item.name, e)
		}
	}
	return nil
}

func NewComponent(name string, init, run ComponentFunc) Component {
	return Component{
		name:   name,
		initFn: init,
		runFn:  run,
	}
}
