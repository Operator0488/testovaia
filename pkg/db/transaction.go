package db

import (
	"context"
	"fmt"
)

// txCtxKey — ключ контекста для хранения транзакции
type txCtxKey struct{}

// TxFromContext извлекает Querier транзакции из контекста.
// Возвращает nil, если транзакции нет.
func TxFromContext(ctx context.Context) Querier {
	tx, _ := ctx.Value(txCtxKey{}).(Querier)
	return tx
}

// contextWithTx помещает транзакцию в контекст
func contextWithTx(ctx context.Context, tx Querier) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// WithTransaction выполняет fn в транзакции.
// Транзакция помещается в контекст — репозитории, использующие BaseRepository.Querier(ctx),
// автоматически будут выполнять запросы в этой транзакции.
// Возвращает ErrNestedTransaction, если в контексте уже есть транзакция.
// Если fn возвращает nil — commit, иначе — rollback.
func (m *manager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	if TxFromContext(ctx) != nil {
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
