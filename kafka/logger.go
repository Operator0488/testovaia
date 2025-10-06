package kafka

import (
	"context"
	"fmt"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

type loggerWrap struct {
	errors bool
}

func (l *loggerWrap) Printf(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	if l.errors {
		logger.Error(context.TODO(), msg)
		return
	}
	logger.Info(context.TODO(), msg)

}
