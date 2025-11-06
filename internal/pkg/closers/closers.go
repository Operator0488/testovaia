// closers provides tools for accumulating and correctly closing components
package closers

import (
	"container/list"
	"context"
	"errors"
	"sync"
)

type CloserFunc func() error

type closer struct {
	list list.List
	m    sync.Mutex
}

type Closer interface {
	Add(item CloserFunc)
	Close(context.Context) error
}

func New() Closer {
	return &closer{}
}

func (c *closer) Add(item CloserFunc) {
	c.m.Lock()
	defer c.m.Unlock()
	c.list.PushFront(item)
}

func (c *closer) Close(ctx context.Context) error {
	c.m.Lock()
	defer c.m.Unlock()
	res := make(chan error, 1)
	closeAll := func() {
		var err error
		for e := c.list.Front(); e != nil; e = e.Next() {
			closer := e.Value.(CloserFunc)
			closeErr := closer()
			err = errors.Join(err, closeErr)
		}
		res <- err
		close(res)
	}

	go closeAll()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-res:
		return err
	}
}
