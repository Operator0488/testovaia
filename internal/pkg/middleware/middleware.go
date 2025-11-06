package middleware

import (
	"net/http"
	"sync"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

type middlewares struct {
	list []Middleware
	mu   *sync.Mutex
}

type Middlewares interface {
	Chain() Middleware
	Add(Middleware)
}

func New() Middlewares {
	return &middlewares{
		mu: &sync.Mutex{},
	}
}

// Chain create invoke chain for HandlerFunc.
func (m *middlewares) Chain() Middleware {
	return func(final http.HandlerFunc) http.HandlerFunc {
		m.mu.Lock()
		defer m.mu.Unlock()
		for i := len(m.list) - 1; i >= 0; i-- {
			final = m.list[i](final)
		}
		return final
	}
}

// Add new middleware to list.
func (m *middlewares) Add(middleware Middleware) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.list = append(m.list, middleware)
}
