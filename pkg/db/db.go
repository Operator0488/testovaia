package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// manager - менеджер подключений к PostgreSQL
type manager struct {
	config Config
	pool   *pgxpool.Pool

	// Health check
	healthStatus    bool
	healthErr       error
	stopHealthCheck chan struct{}
}

// Manager — внутренний интерфейс для управления подключением
type Manager interface {
	Connect(context.Context) error
	Migrate(context.Context) (int, error)
	Close() error
	HealthStatus() (bool, error)
	DB() DbClient
}

// DbClient — публичный интерфейс для сервисов (через DI).
type DbClient interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	// Pool возвращает *pgxpool.Pool для операций, которым нужен именно пул (CopyFrom, Acquire и т.д.)
	Pool() *pgxpool.Pool
}

// NewManager создает новый менеджер подключений
func NewPostgresManager(ctx context.Context, cfg Config) (Manager, error) {
	manager := &manager{
		config:          cfg,
		stopHealthCheck: make(chan struct{}),
	}

	return manager, nil
}

// IsHealthy возвращает true если соединение здорово
func (m *manager) IsHealthy() bool {
	status, _ := m.HealthStatus()
	return status
}

func (m *manager) DB() DbClient {
	return m
}

func (m *manager) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.Exec(ctx, sql, args...)
	}
	return m.pool.Exec(ctx, sql, args...)
}

func (m *manager) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.Query(ctx, sql, args...)
	}
	return m.pool.Query(ctx, sql, args...)
}

func (m *manager) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryRow(ctx, sql, args...)
	}
	return m.pool.QueryRow(ctx, sql, args...)
}

// Pool возвращает *pgxpool.Pool для операций, требующих прямой доступ к пулу
func (m *manager) Pool() *pgxpool.Pool {
	return m.pool
}

// NewDbClient создаёт DbClient из готового пула соединений.
func NewDbClient(pool *pgxpool.Pool) DbClient {
	return &manager{pool: pool, stopHealthCheck: make(chan struct{})}
}
