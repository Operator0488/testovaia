package db

import (
	"context"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"go.uber.org/zap"
)

// startHealthCheck запускает фоновую проверку доступности соединения
func (m *manager) startHealthCheck() {
	go func() {
		ticker := time.NewTicker(m.config.HealthCheckInterval)
		defer ticker.Stop()
		ctx := context.Background()

		for {
			select {
			case <-ticker.C:
				m.checkHealth(ctx)
			case <-m.stopHealthCheck:
				return
			}
		}
	}()
}

// checkHealth проверяет здоровье соединения
func (m *manager) checkHealth(ctx context.Context) {
	sqlDB, err := m.db.DB()
	if err != nil {
		m.healthStatus = false
		m.healthErr = err
		logger.Error(ctx, "health check failed to get sql.DB", zap.Error(err))
		return
	}

	if err := sqlDB.Ping(); err != nil {
		m.healthStatus = false
		m.healthErr = err
		logger.Warn(ctx, "postgres health check failed", zap.Error(err))
	} else {
		m.healthStatus = true
		m.healthErr = nil
	}
}

// HealthStatus возвращает текущий статус здоровья
func (m *manager) HealthStatus() (bool, error) {
	return m.healthStatus, m.healthErr
}
