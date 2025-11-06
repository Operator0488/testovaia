package kafka

import (
	"context"
	"sync/atomic"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

type HealthState struct {
	err       error
	lastCheck time.Time
}

type healthChecker struct {
	healthChannel chan error
	checkInterval time.Duration
	forceCheck    func(ctx context.Context) error
	state         atomic.Value
}

type HealthCheker interface {
	Run(ctx context.Context)
	GetState() HealthState
	Update(err error)
}

// newHealthChecker create component for checking service
func newHealthChecker(
	checkInterval time.Duration,
	forceCheck func(ctx context.Context) error,
) HealthCheker {
	h := &healthChecker{
		healthChannel: make(chan error, 1),
		forceCheck:    forceCheck,
		checkInterval: checkInterval,
	}
	h.updateState(nil)
	return h
}

func (h *healthChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(h.healthChannel)
			return
		case <-ticker.C:
			state := h.GetState()
			if time.Since(state.lastCheck) > h.checkInterval {
				err := h.forceCheck(ctx)
				logger.Debug(ctx, "health check force updated", logger.Err(err))
				h.updateState(err)
			}
		case v := <-h.healthChannel:
			logger.Debug(ctx, "health check updated from channel", logger.Err(v))
			h.updateState(v)
		}
	}
}

func (h *healthChecker) updateState(err error) {
	h.state.Store(&HealthState{
		err:       err,
		lastCheck: time.Now(),
	})
}

func (h *healthChecker) GetState() HealthState {
	s := h.state.Load()
	if s == nil {
		return HealthState{lastCheck: time.Now()}
	}
	return *s.(*HealthState)
}

func (h *healthChecker) Update(err error) {
	select {
	case h.healthChannel <- err:
	default:
	}
}
