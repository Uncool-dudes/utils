package resilience

import (
	"time"

	"github.com/sony/gobreaker/v2"
)

// CircuitBreaker wraps sony/gobreaker/v2 with a domain-friendly API.
type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

// CBConfig controls circuit breaker behavior.
type CBConfig struct {
	Name string

	// MaxRequests is the max number of probe requests allowed in half-open state (default 1).
	MaxRequests uint32
	// Interval is the rolling window for the closed state. 0 disables periodic reset.
	Interval time.Duration
	// Timeout is how long the breaker stays open before transitioning to half-open (default 60s).
	Timeout time.Duration

	// ConsecutiveFailures trips the breaker after N consecutive failures (default 5).
	// Ignored if ReadyToTrip is provided.
	ConsecutiveFailures uint32

	// ReadyToTrip overrides ConsecutiveFailures with full access to Counts.
	// Return true to trip the breaker.
	ReadyToTrip func(counts gobreaker.Counts) bool

	// OnStateChange is called on every state transition. Wire zap logging here.
	OnStateChange func(name string, from, to gobreaker.State)
}

// CBDefaults provides conservative circuit breaker defaults.
var CBDefaults = CBConfig{
	MaxRequests:         1,
	Timeout:             60 * time.Second,
	ConsecutiveFailures: 5,
}

// NewCircuitBreaker creates a CircuitBreaker from cfg.
func NewCircuitBreaker(cfg CBConfig) *CircuitBreaker {
	readyToTrip := cfg.ReadyToTrip
	if readyToTrip == nil {
		threshold := cfg.ConsecutiveFailures
		if threshold == 0 {
			threshold = CBDefaults.ConsecutiveFailures
		}
		readyToTrip = func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= threshold
		}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = CBDefaults.Timeout
	}

	maxRequests := cfg.MaxRequests
	if maxRequests == 0 {
		maxRequests = CBDefaults.MaxRequests
	}

	cb := gobreaker.NewCircuitBreaker[any](gobreaker.Settings{
		Name:          cfg.Name,
		MaxRequests:   maxRequests,
		Interval:      cfg.Interval,
		Timeout:       timeout,
		ReadyToTrip:   readyToTrip,
		OnStateChange: cfg.OnStateChange,
	})

	return &CircuitBreaker{cb: cb}
}

// Execute runs fn through the circuit breaker.
func (c *CircuitBreaker) Execute(fn func() (any, error)) (any, error) {
	return c.cb.Execute(fn) //nolint:wrapcheck // gobreaker errors pass through fn or are sentinels
}

// State returns the current circuit breaker state.
func (c *CircuitBreaker) State() gobreaker.State {
	return c.cb.State()
}

// Name returns the circuit breaker name.
func (c *CircuitBreaker) Name() string {
	return c.cb.Name()
}
