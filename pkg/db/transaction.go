package db

import (
	"context"

	"gorm.io/gorm"
)

// contextKey - кастомный тип для ключа контекста (безопасный от коллизий)
type contextKey string

const (
	transactionKey contextKey = "gorm_tx"
)

// withTransactionContext помещает транзакцию в контекст
func withTransactionContext(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, transactionKey, tx)
}

// transactionFromContext извлекает транзакцию из контекста
func transactionFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(transactionKey).(*gorm.DB)
	return tx, ok
}

// WithTransaction выполняет операцию в транзакции
func (m *manager) WithTransaction(ctx context.Context, fn func(txContext context.Context) error) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		ctx = withTransactionContext(ctx, tx)
		return fn(ctx)
	})
}

// DB возвращает инициализированный инстанс *gorm.DB
// Если в переданном контексте уже есть транзакционный инстанс то вернется он
func (m *manager) DB(ctx context.Context) *gorm.DB {
	if tx, ok := transactionFromContext(ctx); ok {
		return tx
	}
	return m.db
}
