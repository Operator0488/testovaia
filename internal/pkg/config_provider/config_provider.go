package configprovider

import (
	"context"
	"errors"
)

var (
	ErrUnsupported = errors.New("method unsupported")
)

//go:generate go run go.uber.org/mock/mockgen -destination=mock/mock.go -package=mock -source=config_provider.go

// Provider for configurations: vault, consul etc.
type Provider interface {
	Get(ctx context.Context) (ConfigData, error)
	Set(ctx context.Context, value ConfigData) error
	Watch(ctx context.Context, onChange func(map[string]interface{})) error
	Close(ctx context.Context) error
}

type ConfigData map[string]interface{}
