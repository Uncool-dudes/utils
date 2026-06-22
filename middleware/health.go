package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/alexliesenfeld/health"
	"go.uber.org/zap"
)

// Check is re-exported so callers don't need to import alexliesenfeld/health directly.
type Check = health.Check

// ReadinessOption configures NewReadinessHandler.
type ReadinessOption func(*readinessCfg)

type readinessCfg struct {
	checks []health.Check
	log    *zap.Logger
}

// WithCheck adds a dependency check to /readyz.
func WithCheck(c Check) ReadinessOption {
	return func(cfg *readinessCfg) { cfg.checks = append(cfg.checks, c) }
}

// WithLogger adds a status-change listener that logs Error when any check flips to down.
// Error-level logs flow to Sentry automatically when sentry.Module is wired.
func WithLogger(log *zap.Logger) ReadinessOption {
	return func(cfg *readinessCfg) { cfg.log = log }
}

// NewLivenessHandler returns an HTTP handler for /healthz.
// Always returns 200 — signals the process is alive, not that dependencies are ready.
func NewLivenessHandler() http.Handler {
	return health.NewHandler(health.NewChecker())
}

// NewReadinessHandler returns an HTTP handler for /readyz.
// Results are cached for 5s to protect dependencies from probe storms.
// Returns 200 when all checks pass, 503 with JSON component detail when any fail.
//
//	middleware.NewReadinessHandler(
//	    middleware.WithLogger(log),
//	    middleware.WithCheck(middleware.Check{Name: "database", Check: svc.HealthCheck, Timeout: 2*time.Second}),
//	)
func NewReadinessHandler(opts ...ReadinessOption) http.Handler {
	cfg := &readinessCfg{}
	for _, o := range opts {
		o(cfg)
	}

	checkerOpts := make([]health.CheckerOption, 0, 3+len(cfg.checks))
	checkerOpts = append(checkerOpts,
		health.WithCacheDuration(5*time.Second),
		health.WithTimeout(10*time.Second),
	)
	for _, c := range cfg.checks {
		checkerOpts = append(checkerOpts, health.WithCheck(c))
	}
	if cfg.log != nil {
		log := cfg.log
		checkerOpts = append(checkerOpts, health.WithStatusListener(
			func(_ context.Context, state health.CheckerState) {
				for name, cs := range state.CheckState {
					if cs.Status == health.StatusDown {
						log.Error("dependency down",
							zap.String("check", name),
							zap.Error(cs.Result),
						)
					}
				}
			},
		))
	}
	return health.NewHandler(health.NewChecker(checkerOpts...))
}
