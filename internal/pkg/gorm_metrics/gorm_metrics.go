package gormmetrics

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

// GormMetrics представляет коллектор метрик для GORM
type GormMetrics struct {
	namespace string
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Метрики
	queryDuration    *prometheus.HistogramVec
	queryTotal       *prometheus.CounterVec
	errorTotal       *prometheus.CounterVec
	connectionStatus *prometheus.GaugeVec
	connectionStats  *prometheus.GaugeVec
}

// Config конфигурация для метрик
type Config struct {
	Namespace       string
	Subsystem       string
	DurationBuckets []float64
	Registerer      prometheus.Registerer
	Context         context.Context
	MonitorInterval time.Duration
}

// New создает новый экземпляр GormMetrics
func New(cfg Config) (*GormMetrics, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "gorm"
	}
	if cfg.Subsystem == "" {
		cfg.Subsystem = "db"
	}
	if cfg.DurationBuckets == nil {
		cfg.DurationBuckets = prometheus.DefBuckets
	}
	if cfg.Registerer == nil {
		cfg.Registerer = prometheus.DefaultRegisterer
	}
	if cfg.Context == nil {
		cfg.Context = context.Background()
	}
	if cfg.MonitorInterval == 0 {
		cfg.MonitorInterval = 30 * time.Second
	}

	namespace := cfg.Namespace

	ctx, cancel := context.WithCancel(cfg.Context)

	metrics := &GormMetrics{
		namespace: namespace,
		ctx:       ctx,
		cancel:    cancel,
		queryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "query_duration_seconds",
				Help:      "Duration of GORM queries in seconds",
				Buckets:   cfg.DurationBuckets,
			},
			[]string{"operation", "table", "success"},
		),
		queryTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "queries_total",
				Help:      "Total number of GORM queries",
			},
			[]string{"operation", "table"},
		),
		errorTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "Total number of GORM query errors",
			},
			[]string{"operation", "table"},
		),
		connectionStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "connection_status",
				Help:      "Database connection status (1 = connected, 0 = disconnected)",
			},
			[]string{},
		),
		connectionStats: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "connection_pool_stats",
				Help:      "Database connection pool statistics",
			},
			[]string{"stat"},
		),
	}

	// Регистрируем метрики
	if err := metrics.registerAll(
		cfg.Registerer,
		metrics.queryDuration,
		metrics.queryTotal,
		metrics.errorTotal,
		metrics.connectionStatus,
		metrics.connectionStats,
	); err != nil {
		return nil, err
	}

	return metrics, nil
}

func (m *GormMetrics) registerAll(registerer prometheus.Registerer, collectors ...prometheus.Collector) (err error) {
	for _, collector := range collectors {
		if cerr := registerer.Register(collector); cerr != nil {
			err = errors.Join(cerr)
		}
	}
	return
}

// Name возвращает имя плагина (реализация gorm.Plugin интерфейса)
func (m *GormMetrics) Name() string {
	return "prometheus_metrics"
}

// Initialize инициализирует плагин сбора метрик в gorm.DB
func (m *GormMetrics) Initialize(db *gorm.DB) error {
	m.registerCallbacks(db)

	// Настраиваем отслеживание статуса подключения
	m.monitorConnection(db)

	return db.Use(m)
}

// registerCallbacks регистрирует callback'и для операций GORM
func (m *GormMetrics) registerCallbacks(db *gorm.DB) {
	operations := []struct {
		name   string
		before func(name string, fn func(*gorm.DB)) error
		after  func(name string, fn func(*gorm.DB)) error
	}{
		{
			name:   "query",
			before: db.Callback().Query().Before("gorm:query").Register,
			after:  db.Callback().Query().After("gorm:query").Register,
		},
		{
			name:   "create",
			before: db.Callback().Create().Before("gorm.create").Register,
			after:  db.Callback().Create().After("gorm.create").Register,
		},
		{
			name:   "update",
			before: db.Callback().Update().Before("gorm.update").Register,
			after:  db.Callback().Update().After("gorm.update").Register,
		},
		{
			name:   "delete",
			before: db.Callback().Delete().Before("gorm.delete").Register,
			after:  db.Callback().Delete().After("gorm.delete").Register,
		},
		{
			name:   "row",
			before: db.Callback().Row().Before("gorm.row").Register,
			after:  db.Callback().Row().After("gorm.row").Register,
		},
		{
			name:   "raw",
			before: db.Callback().Row().Before("gorm.raw").Register,
			after:  db.Callback().Row().After("gorm.raw").Register,
		},
	}

	for _, operation := range operations {
		name := operation.name
		startTime := "start_time_" + name
		operation.before("prometheus:before_"+name, func(d *gorm.DB) {
			db.Set(startTime, time.Now())
		})

		operation.after("prometheus:after_"+name, func(d *gorm.DB) {
			startTime, exists := db.Get(startTime)
			if !exists {
				return
			}

			duration := time.Since(startTime.(time.Time)).Seconds()
			table := m.getTableName(db)
			success := strconv.FormatBool(db.Error == nil)

			m.queryTotal.WithLabelValues(name, table).Inc()
			m.queryDuration.WithLabelValues(name, table, success).Observe(duration)

			if db.Error != nil {
				m.errorTotal.WithLabelValues(name, table).Inc()
			}
		})
	}

}

// monitorConnection отслеживает статус подключения и статистику пула
func (m *GormMetrics) monitorConnection(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				// Устанавливаем статус "отключено" при завершении
				m.connectionStatus.WithLabelValues().Set(0)
				return
			case <-ticker.C:
				m.updateConnectionMetrics(sqlDB)
			}
		}
	}()
}

// updateConnectionMetrics обновляет метрики подключения
func (m *GormMetrics) updateConnectionMetrics(sqlDB *sql.DB) {
	if err := sqlDB.Ping(); err == nil {
		m.connectionStatus.WithLabelValues().Set(1)
	} else {
		m.connectionStatus.WithLabelValues().Set(0)
	}

	// Получаем статистику пула подключений
	stats := sqlDB.Stats()
	m.connectionStats.WithLabelValues("open_connections").Set(float64(stats.OpenConnections))
	m.connectionStats.WithLabelValues("in_use").Set(float64(stats.InUse))
	m.connectionStats.WithLabelValues("idle").Set(float64(stats.Idle))
	m.connectionStats.WithLabelValues("max_open_connections").Set(float64(stats.MaxOpenConnections))
	m.connectionStats.WithLabelValues("wait_count").Set(float64(stats.WaitCount))
	m.connectionStats.WithLabelValues("wait_duration").Set(float64(stats.WaitDuration))
	m.connectionStats.WithLabelValues("max_idle_closed").Set(float64(stats.MaxIdleClosed))
	m.connectionStats.WithLabelValues("max_lifetime_closed").Set(float64(stats.MaxLifetimeClosed))
}

// getTableName извлекает имя таблицы из контекста GORM
func (m *GormMetrics) getTableName(db *gorm.DB) string {
	stmt := db.Statement
	if stmt == nil || stmt.Schema == nil {
		return "unknown"
	}

	if stmt.Schema.Table != "" {
		return stmt.Schema.Table
	}

	if stmt.Table != "" {
		return stmt.Table
	}

	return "unknown"
}

// Use регистрирует плагин в GORM DB
func (m *GormMetrics) Use(db *gorm.DB) error {
	return db.Use(m)
}

// Close останавливает мониторинг и освобождает ресурсы
func (m *GormMetrics) Close() error {
	if m.cancel != nil {
		m.cancel()
		m.wg.Wait() // Ждем завершения всех горутин
	}
	return nil
}
