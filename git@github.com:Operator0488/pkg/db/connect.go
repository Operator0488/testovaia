package db

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (m *manager) Connect(ctx context.Context) error {
	poolConfig, err := pgxpool.ParseConfig(m.buildDSN())
	if err != nil {
		return fmt.Errorf("parse pool config: %w", err)
	}
	poolConfig.MaxConnLifetime = m.config.ConnMaxLifetime
	poolConfig.MaxConns = int32(m.config.MaxOpenConns)
	poolConfig.MinConns = int32(m.config.MaxIdleConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping: %w", err)
	}

	m.pool = pool
	m.startHealthCheck()

	return nil
}

// Close закрывает соединение и останавливает health check
func (m *manager) Close() error {
	close(m.stopHealthCheck)

	if m.pool != nil {
		m.pool.Close()
	}

	return nil
}

// buildDSN строит DSN строку
func (m *manager) buildDSN() string {
	if m.config.DSN != "" {
		return m.config.DSN
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(m.config.User, m.config.Password),
		Host:   fmt.Sprintf("%s:%d", m.config.Host, m.config.Port),
		Path:   m.config.Database,
	}

	query := u.Query()
	query.Set("sslmode", m.config.SSLMode)
	u.RawQuery = query.Encode()

	return u.String()
}
