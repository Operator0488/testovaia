package grpc

import (
	"context"
	"fmt"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"google.golang.org/grpc/grpclog"
	"os"
	"strconv"
	"strings"
	"sync"
)

type severity int

const (
	sevInfo severity = iota
	sevWarning
	sevError
	sevFatal
)

func parseSeverity(s string) severity {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "INFO":
		return sevInfo
	case "WARNING", "WARN":
		return sevWarning
	case "ERROR":
		return sevError
	case "FATAL":
		return sevFatal
	default:
		// WARNING и выше
		return sevWarning
	}
}

type grpcLogger struct {
	ctx         context.Context
	minSeverity severity
	verbosity   int
}

func (l grpcLogger) Info(args ...any) { logger.Info(l.ctx, fmt.Sprint(args...)) }
func (l grpcLogger) Infoln(args ...any) {
	logger.Info(l.ctx, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}
func (l grpcLogger) Infof(format string, args ...any) {
	logger.Info(l.ctx, fmt.Sprintf(format, args...))
}

func (l grpcLogger) Warning(args ...any) { logger.Warn(l.ctx, fmt.Sprint(args...)) }
func (l grpcLogger) Warningln(args ...any) {
	logger.Warn(l.ctx, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}
func (l grpcLogger) Warningf(format string, args ...any) {
	logger.Warn(l.ctx, fmt.Sprintf(format, args...))
}

func (l grpcLogger) Error(args ...any) { logger.Error(l.ctx, fmt.Sprint(args...)) }
func (l grpcLogger) Errorln(args ...any) {
	logger.Error(l.ctx, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}
func (l grpcLogger) Errorf(format string, args ...any) {
	logger.Error(l.ctx, fmt.Sprintf(format, args...))
}

func (l grpcLogger) Fatal(args ...any) { logger.Fatal(l.ctx, fmt.Sprint(args...)); os.Exit(1) }
func (l grpcLogger) Fatalln(args ...any) {
	logger.Fatal(l.ctx, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
	os.Exit(1)
}

func (l grpcLogger) Fatalf(format string, args ...any) {
	logger.Fatal(l.ctx, fmt.Sprintf(format, args...))
	os.Exit(1)
}
func (l grpcLogger) V(level int) bool { return level <= l.verbosity }

var setOnce sync.Once

// EnableWithContext нужно вызвать ДО grpc.Dial()/grpc.NewServer()
func EnableWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	minSev := parseSeverity(os.Getenv("log.level"))

	verb := 0
	if s := os.Getenv("log.level"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			verb = n
		}
	}

	setOnce.Do(func() {
		grpclog.SetLoggerV2(grpcLogger{
			ctx:         ctx,
			minSeverity: minSev,
			verbosity:   verb,
		})
	})
}
