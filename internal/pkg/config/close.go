package config

import (
	"context"
	"errors"
)

// Close stop watchers.
func (c *Config) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	var err error
	for _, storage := range c.storages {
		perr := storage.provider.Close(ctx)
		if perr != nil {
			err = errors.Join(err, perr)
		}
	}

	if c.watchChan != nil {
		close(c.watchChan)
	}

	c.closed = true
	return err
}
