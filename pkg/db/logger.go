package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

type gormLogger struct {
	Logger                    logger.Logger
	LogLevel                  glogger.LogLevel
	SlowThreshold             time.Duration
	Parameterized             bool
	IgnoreRecordNotFoundError bool
}

func newLogger(logger logger.Logger, config Config) glogger.Interface {
	return &gormLogger{
		Logger:                    logger,
		LogLevel:                  getLogLevel(config.LogLevel),
		SlowThreshold:             config.SlowThreshold,
		Parameterized:             config.parameterizedQueries,
		IgnoreRecordNotFoundError: config.ignoreRecordNotFoundError,
	}
}

func (l *gormLogger) LogMode(level glogger.LogLevel) glogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *gormLogger) withComponent(ctx context.Context) context.Context {
	return l.Logger.With(ctx, logger.String("component", "gorm"))
}

func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= glogger.Info {
		l.Logger.Info(l.withComponent(ctx), msg, logger.Any("data", data))
	}
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= glogger.Warn {
		l.Logger.Warn(l.withComponent(ctx), msg, logger.Any("data", data))
	}
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= glogger.Error {
		l.Logger.Error(l.withComponent(ctx), msg, logger.Any("data", data))
	}
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= glogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []logger.Field{
		logger.String("duration", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)),
		logger.String("sql", sql),
	}

	if rows != -1 {
		fields = append(fields, logger.Int64("rows", rows))
	}

	switch {
	case err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		fields = append(fields, logger.Err(err))
		l.Logger.Error(l.withComponent(ctx), "SQL executed", fields...)

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold:
		l.Logger.Warn(l.withComponent(ctx), "SQL executed", fields...)

	case l.LogLevel >= glogger.Info:
		l.Logger.Info(l.withComponent(ctx), "SQL executed", fields...)
	}
}

// ParamsFilter filter params
func (l *gormLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Parameterized {
		return sql, nil
	}
	return sql, params
}

func getLogLevel(level string) glogger.LogLevel {
	logLevel := strings.ToLower(level)
	switch logLevel {
	case "debug", "info":
		return glogger.Info
	case "warn":
		return glogger.Warn
	case "error":
		return glogger.Error
	default:
		return glogger.Silent
	}
}
