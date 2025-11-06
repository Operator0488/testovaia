package db

import (
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/config"
)

// Config - конфигурация подключения к PostgreSQL
type Config struct {
	// Готовая DSN строка
	DSN string

	// Отдельные параметры для сборки DSN
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	// Настройки пула соединений
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration

	// Настройки логирования
	LogLevel      string
	SlowThreshold time.Duration

	// Health check
	HealthCheckInterval time.Duration

	parameterizedQueries bool

	ignoreRecordNotFoundError bool

	migrateRetries int
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		Host:                      "localhost",
		Port:                      5432,
		User:                      "app_user",
		Password:                  "app_password",
		Database:                  "postgres",
		SSLMode:                   "disable",
		MaxOpenConns:              25,
		MaxIdleConns:              25,
		ConnMaxLifetime:           5 * time.Minute,
		LogLevel:                  "info",
		SlowThreshold:             200 * time.Millisecond,
		HealthCheckInterval:       30 * time.Second,
		parameterizedQueries:      false,
		ignoreRecordNotFoundError: true,
		migrateRetries:            3,
	}
}

// LoadConfig загружает конфигурацию из Configurer
func LoadConfig(cfg config.Configurer) Config {
	config := DefaultConfig()

	// Приоритет: готовая DSN строка
	if dsn := cfg.GetString("postgres.dsn"); dsn != "" {
		config.DSN = dsn
	} else {
		// Собираем DSN из отдельных параметров
		config.Host = cfg.GetStringOrDefault("postgres.host", config.Host)
		config.Port = cfg.GetIntOrDefault("postgres.port", config.Port)
		config.User = cfg.GetStringOrDefault("postgres.user", config.User)
		config.Password = cfg.GetStringOrDefault("postgres.password", config.Password)
		config.Database = cfg.GetStringOrDefault("postgres.database", config.Database)
		config.SSLMode = cfg.GetStringOrDefault("postgres.sslmode", config.SSLMode)
	}

	// Настройки пула соединений
	if maxOpenConns := cfg.GetInt("postgres.max_open_conns"); maxOpenConns > 0 {
		config.MaxOpenConns = maxOpenConns
	}

	if maxIdleConns := cfg.GetInt("postgres.max_idle_conns"); maxIdleConns > 0 {
		config.MaxIdleConns = maxIdleConns
	}

	if connMaxLifetime := cfg.GetDuration("postgres.conn_max_lifetime"); connMaxLifetime > 0 {
		config.ConnMaxLifetime = connMaxLifetime
	}

	// Настройки логирования
	config.LogLevel = cfg.GetStringOrDefault("LOG_LEVEL", config.LogLevel)

	if slowThreshold := cfg.GetDuration("postgres.slow_threshold"); slowThreshold > 0 {
		config.SlowThreshold = slowThreshold
	}

	// Health check
	if healthCheckInterval := cfg.GetDuration("postgres.health_check_interval"); healthCheckInterval > 0 {
		config.HealthCheckInterval = healthCheckInterval
	}

	return config
}
