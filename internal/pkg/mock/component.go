package mock

import (
	"context"
)

//go:generate go run go.uber.org/mock/mockgen -destination=mock.go -package=mock -source=component.go

type TestComponent interface {
	Init(ctx context.Context) error
	Run(ctx context.Context) error
	HealthCheck(ctx context.Context) error
	Close(ctx context.Context) error
}
