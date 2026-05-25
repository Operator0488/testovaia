package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// txCtxKey — ключ контекста для хранения транзакции
type txCtxKey struct{}

// txFromContext извлекает транзакцию из контекста.
// Возвращает nil, если транзакции нет.
func txFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txCtxKey{}).(pgx.Tx)
	return tx
}

// contextWithTx помещает транзакцию в контекст
func contextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// WithTransaction выполняет fn в транзакции.
// Транзакция помещается в контекст — методы Exec/Query/QueryRow автоматически
// выполняют запросы в этой транзакции.
// Возвращает ErrNestedTransaction, если в контексте уже есть транзакция.
// Если fn возвращает nil — commit, иначе — rollback.
func (m *manager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	if txFromContext(ctx) != nil {
		return ErrNestedTransaction
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op после commit

	if err := fn(contextWithTx(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
