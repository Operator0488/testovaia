package logger

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"testing"
)

func TestLogger_With(t *testing.T) {
	const (
		testMsg      = "test message"
		testKey      = "fieldKey"
		testKeyValue = "field value"
	)

	log, observedLogs := resolveMockLogger()

	ctx := context.Background()
	ctx = log.With(ctx, zap.String(testKey, testKeyValue))

	log.Debug(ctx, testMsg)
	logEntry := getFirstLogEntry(t, observedLogs)
	if logEntry.Message != testMsg {
		t.Errorf("Expected message '%s', got: %s", testMsg, logEntry.Message)
	}

	// Проверка наличия добавленного поля в структурном логе
	for _, field := range logEntry.Context {
		if field.Key == testKey && field.Type == zapcore.StringType && field.String == testKeyValue {
			return
		}
	}

	t.Errorf("Expected log to contain field '%s=%s', got: %v", testKey, testKeyValue, logEntry.Context)
}

func TestLogger_WithTraceID(t *testing.T) {

	const traceIdValue = "4bf92f3577b34da6a3ce929d0e0e4736"

	log, observedLogs := resolveMockLogger()
	traceID, err := trace.TraceIDFromHex(traceIdValue)
	if err != nil {
		t.Fatalf("Failed to create traceID: %v", err)
	}
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	log.Debug(ctx, "some text")
	logEntry := getFirstLogEntry(t, observedLogs)
	for _, field := range logEntry.Context {
		if field.Key == traceId && field.Type == zapcore.StringType && field.String == traceIdValue {
			return
		}
	}

	t.Errorf("Expected log to contain field 'traceId=%s', got: %v", traceIdValue, logEntry.Context)
}

func resolveMockLogger() (*logger, *observer.ObservedLogs) {
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	return &logger{observedLogger}, observedLogs
}

func getFirstLogEntry(t *testing.T, o *observer.ObservedLogs) observer.LoggedEntry {
	if o.Len() == 0 {
		t.Fatal("No logs recorded")
	}

	return o.All()[0]
}
