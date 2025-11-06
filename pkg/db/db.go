package db

import (
	"context"
	"fmt"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"gorm.io/gorm"
)

// Custom errors
var (
	ErrConnectionFailed  = fmt.Errorf("postgres connection failed")
	ErrNotConnected      = fmt.Errorf("postgres not connected")
	ErrHealthCheckFailed = fmt.Errorf("postgres health check failed")
)

// manager - менеджер подключений к PostgreSQL
type manager struct {
	config Config
	db     *gorm.DB

	// Health check
	healthStatus    bool
	healthErr       error
	stopHealthCheck chan struct{}
}

type Manager interface {
	DB(context.Context) *gorm.DB
	Connect(context.Context) error
	Migrate(context.Context) (int, error)
	Close() error
	WithTransaction(context.Context, func(context.Context) error) error
	HealthStatus() (bool, error)
}

type DbClient interface {
	DB(context.Context) *gorm.DB
	WithTransaction(context.Context, func(context.Context) error) error
}

// NewManager создает новый менеджер подключений
func NewPostgresManager(ctx context.Context, cfg Config) (Manager, error) {
	manager := &manager{
		config:          cfg,
		stopHealthCheck: make(chan struct{}),
	}

	return manager, nil
}

// Close закрывает соединение и останавливает health check
func (m *manager) Close() error {
	close(m.stopHealthCheck)

	if m.db != nil {
		sqlDB, err := m.db.DB()
		if err != nil {
			return fmt.Errorf("failed to get sql.DB: %w", err)
		}
		return sqlDB.Close()
	}

	return nil
}

// IsHealthy возвращает true если соединение здорово
func (m *manager) IsHealthy() bool {
	status, _ := m.HealthStatus()
	return status
}

// configureConnectionPool настраивает пул соединений
func (m *manager) configureConnectionPool(ctx context.Context) {
	sqlDB, err := m.db.DB()
	if err != nil {
		logger.Error(ctx, "failed to get sql.DB from gorm", logger.Err(err))
		return
	}

	sqlDB.SetMaxOpenConns(m.config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(m.config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(m.config.ConnMaxLifetime)
	//sqlDB.SetConnMaxIdleTime(time.Minute * 5) // можно добавить
}
