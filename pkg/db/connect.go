package db

import (
	"context"
	"fmt"
	"net/url"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func (m *manager) connect() error {
	dsn := m.buildDSN()

	gormConfig := &gorm.Config{}

	// Настройка логирования GORM в зависимости от уровня
	if m.config.LogLevel != "none" {
		gormConfig.Logger = newLogger(logger.GetLogger(), m.config)
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to open gorm connection: %w", err)
	}

	m.db = db
	return nil
}

func (m *manager) ping(ctx context.Context) error {
	db, err := m.db.DB()
	if err != nil {
		return err
	}
	return db.PingContext(ctx)
}

func (m *manager) Connect(ctx context.Context) error {
	if err := m.connect(); err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := m.ping(ctx); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	m.configureConnectionPool(ctx)
	m.startHealthCheck()

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
