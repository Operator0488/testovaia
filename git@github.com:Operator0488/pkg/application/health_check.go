package application

import (
	"time"
)

const (
	defaultLivenessTimeout  = time.Second * 30
	defaultReadinessTimeout = time.Second * 5
)

// addProbes add liveness and readiness probes
func (a *Application) addProbes() {
	a.middlewares.Add(a.livenessMiddleware)
	a.middlewares.Add(a.readinessMiddleware)
	a.components.add(component(httpServer))
}
