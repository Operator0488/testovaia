package application

import (
	"errors"
	"fmt"
)

type Option func(app *Application) error

var (
	ErrComponentAlreadyExist = errors.New("component already exist")
)

func WithComponent(name string, init, run ComponentFunc) Option {
	return func(app *Application) error {
		if !app.components.add(component{name: name, initFn: init, runFn: run}) {
			return fmt.Errorf("%w: %s", ErrComponentAlreadyExist, name)
		}
		return nil
	}
}
