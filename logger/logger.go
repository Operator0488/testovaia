package logger

import (
	"context"
	"log"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level = zapcore.Level

type Logger interface {
	Level() Level
	With(ctx context.Context, fields ...Field) context.Context
	Debug(ctx context.Context, msg string, fields ...Field) Logger
	Info(ctx context.Context, msg string, fields ...Field) Logger
	Warn(ctx context.Context, msg string, fields ...Field) Logger
	Error(ctx context.Context, msg string, fields ...Field) Logger
	Fatal(ctx context.Context, msg string, fields ...Field)
}

// With Добавляет постоянную переменную(или переменные) к выводу лога в пределах использования полученного в результате
// вызова функции контекста. Постоянные переменные не будут добавляться к выводу лога, если использовать отличный от
// полученного в результате вызова функции контекст
func With(ctx context.Context, fields ...Field) context.Context {
	return singleInstance.With(ctx, fields...)
}

// Debug Добаялет сообщение в лог с уровнем Debug. Сообщение будет включать в себя все добавленные в результате With
// переменные, TraceId, а также переменные переданные непосредствено при вызове метода
func Debug(ctx context.Context, msg string, fields ...Field) Logger {
	return singleInstance.Debug(ctx, msg, fields...)
}

// Info Добаялет сообщение в лог с уровнем Info. Сообщение будет включать в себя все добавленные в результате With
// переменные, TraceId, а также переменные переданные непосредствено при вызове метода
func Info(ctx context.Context, msg string, fields ...Field) Logger {
	return singleInstance.Info(ctx, msg, fields...)
}

// Warn Добаялет сообщение в лог с уровнем Warn. Сообщение будет включать в себя все добавленные в результате With
// переменные, TraceId, а также переменные переданные непосредствено при вызове метода
func Warn(ctx context.Context, msg string, fields ...Field) Logger {
	return singleInstance.Warn(ctx, msg, fields...)
}

// Error Добаялет сообщение в лог с уровнем Error. Сообщение будет включать в себя все добавленные в результате With
// переменные, StackTrace, TraceId, а также переменные переданные непосредствено при вызове метода
func Error(ctx context.Context, msg string, fields ...Field) Logger {
	return singleInstance.Error(ctx, msg, fields...)
}

// Fatal Добаялет сообщение в лог с уровнем Fatal и завершает работу приложения.
// Сообщение будет включать в себя все добавленные в результате With переменные, TraceId, а также переменные
// переданные непосредствено при вызове метода
func Fatal(ctx context.Context, msg string, fields ...Field) {
	singleInstance.Fatal(ctx, msg, fields...)
}

const (
	CorrelationId = "correlationId"
	traceId       = "traceId"
	nestedLogger  = "nestedLogger"
)

var singleInstance Logger

type logger struct {
	*zap.Logger
}

func init() {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.DisableStacktrace = true
	loggerConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	loggerConfig.Level = zap.NewAtomicLevelAt(getLogLevel(os.Getenv("LOG_LEVEL")))
	loggerInstance, err := loggerConfig.Build()
	if err != nil {
		log.Fatal("failed to init log", err)
	}

	singleInstance = &logger{loggerInstance}
}

func getLogLevel(level string) zapcore.Level {
	logLevel := strings.ToLower(level)
	switch logLevel {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

/*
p_alex: Предполагается, что logger статически доступен и эта функция более не нужна.

	TODO: Удалить, если не вернемся к формату внедрения логгера как зависимости
	TODO: метод нужен в некоторых кейсах
*/
func GetLogger() Logger {
	return singleInstance
}

func (log *logger) Level() Level {
	return log.Level()
}

func (log *logger) With(ctx context.Context, fields ...Field) context.Context {
	activeLogger := log.resolveActiveLogger(ctx)
	nested := &logger{activeLogger.Logger.With(fields...)}
	return context.WithValue(ctx, nestedLogger, nested)
}

func (log *logger) Debug(ctx context.Context, msg string, fields ...Field) Logger {
	activeLogger := log.resolveActiveLogger(ctx)
	fields = activeLogger.addTraceId(ctx, fields...)
	fields = activeLogger.addCorrelationId(ctx, fields...)
	activeLogger.WithOptions(zap.AddCallerSkip(2)).Debug(msg, fields...)
	return activeLogger
}

func (log *logger) Info(ctx context.Context, msg string, fields ...Field) Logger {
	activeLogger := log.resolveActiveLogger(ctx)
	fields = activeLogger.addTraceId(ctx, fields...)
	fields = activeLogger.addCorrelationId(ctx, fields...)
	activeLogger.WithOptions(zap.AddCallerSkip(2)).Info(msg, fields...)
	return activeLogger
}

func (log *logger) Warn(ctx context.Context, msg string, fields ...Field) Logger {
	activeLogger := log.resolveActiveLogger(ctx)
	fields = activeLogger.addTraceId(ctx, fields...)
	fields = activeLogger.addCorrelationId(ctx, fields...)
	activeLogger.WithOptions(zap.AddCallerSkip(2)).Warn(msg, fields...)
	return activeLogger
}

func (log *logger) Error(ctx context.Context, msg string, fields ...Field) Logger {
	activeLogger := log.resolveActiveLogger(ctx)
	fields = activeLogger.addTraceId(ctx, fields...)
	fields = activeLogger.addCorrelationId(ctx, fields...)
	activeLogger.WithOptions(zap.AddCallerSkip(2)).Error(msg, fields...)
	return activeLogger
}

func (log *logger) Fatal(ctx context.Context, msg string, fields ...Field) {
	activeLogger := log.resolveActiveLogger(ctx)
	fields = activeLogger.addTraceId(ctx, fields...)
	fields = activeLogger.addCorrelationId(ctx, fields...)
	activeLogger.WithOptions(zap.AddCallerSkip(2)).Fatal(msg, fields...)
}

func (log *logger) resolveActiveLogger(ctx context.Context) *logger {
	if ctxLogger, ok := ctx.Value(nestedLogger).(*logger); ok {
		return ctxLogger
	}

	return log
}

func (log *logger) addCorrelationId(ctx context.Context, fields ...Field) []Field {
	if ctx != nil {
		corrId := ctx.Value(CorrelationId)
		if corrId != nil {
			return append(fields, zap.String(CorrelationId, corrId.(string)))
		}
	}

	return fields
}

func (log *logger) addTraceId(ctx context.Context, fields ...Field) []Field {
	if ctx != nil {
		if spanCtx := trace.SpanFromContext(ctx).SpanContext(); spanCtx.HasTraceID() {
			traceIdValue := spanCtx.TraceID()
			return append(fields, zap.String(traceId, traceIdValue.String()))
		}
	}

	return fields
}
