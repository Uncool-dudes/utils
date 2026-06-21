package middleware

import (
	"net/http"
	"time"

	"github.com/alexliesenfeld/health"
)

// Check is re-exported so callers don't need to import alexliesenfeld/health directly.
type Check = health.Check

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
//	    middleware.Check{Name: "database", Check: svc.HealthCheck, Timeout: 2*time.Second},
//	)
func NewReadinessHandler(checks ...health.Check) http.Handler {
	opts := make([]health.CheckerOption, 0, 2+len(checks))
	opts = append(opts,
		health.WithCacheDuration(5*time.Second),
		health.WithTimeout(10*time.Second),
	)
	for _, c := range checks {
		opts = append(opts, health.WithCheck(c))
	}
	return health.NewHandler(health.NewChecker(opts...))
}
