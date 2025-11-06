// health provides tools for accumulating and concurrent check components healthy
package health

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

type CheckFunc func(context.Context) error

type Check struct {
	Name string
	Func CheckFunc
}

type health struct {
	checks []Check
	m      sync.RWMutex
}

type Health interface {
	Add(name string, fn CheckFunc)
	Check(ctx context.Context) error
}

func New() Health {
	return &health{}
}

func (c *health) Add(name string, fn CheckFunc) {
	c.m.Lock()
	defer c.m.Unlock()
	c.checks = append(c.checks, Check{Name: name, Func: fn})
}

// Check run async check all registered components
func (c *health) Check(ctx context.Context) error {
	c.m.RLock()
	defer c.m.RUnlock()

	if len(c.checks) == 0 {
		return nil
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, check := range c.checks {
		g.Go(func() error {
			if err := check.Func(ctx); err != nil {
				return &CheckError{
					Name: check.Name,
					Err:  err,
				}
			}
			return nil
		})
	}

	return g.Wait()
}

type CheckError struct {
	Name string
	Err  error
}

func (e *CheckError) Error() string {
	return fmt.Sprintf("check '%s' failed: %v", e.Name, e.Err)
}

func (e *CheckError) Unwrap() error {
	return e.Err
}
