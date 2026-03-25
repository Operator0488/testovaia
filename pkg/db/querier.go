package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier — общий интерфейс для выполнения SQL-запросов.
// Реализуется и *pgxpool.Pool, и pgx.Tx — позволяет репозиториям
// работать одинаково и вне, и внутри транзакции.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TxManager — управление транзакциями.
// Транзакция помещается в контекст; репозитории подхватывают её автоматически
// через BaseRepository.Querier(ctx).
type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
