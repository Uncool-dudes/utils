package resilience

import (
	"context"
	"time"
)

// Policy composes resilience primitives into a single executor.
// Execution order: throttle → timeout → circuit breaker → retry.
type Policy struct {
	cb          *CircuitBreaker
	retry       *RetryConfig
	isRetryable func(error) bool
	throttle    *Throttle
	timeout     time.Duration
}

type PolicyOption func(*Policy)

func WithCircuitBreaker(cfg CBConfig) PolicyOption {
	return func(p *Policy) { p.cb = NewCircuitBreaker(cfg) }
}

// WithRetry enables retries. isRetryable determines which errors are transient.
func WithRetry(cfg RetryConfig, isRetryable func(error) bool) PolicyOption {
	return func(p *Policy) {
		p.retry = &cfg
		p.isRetryable = isRetryable
	}
}

func WithThrottle(cfg ThrottleConfig) PolicyOption {
	return func(p *Policy) { p.throttle = NewThrottle(cfg) }
}

func WithTimeout(d time.Duration) PolicyOption {
	return func(p *Policy) { p.timeout = d }
}

func NewPolicy(opts ...PolicyOption) *Policy {
	p := &Policy{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Execute runs fn through all enabled policy layers.
func (p *Policy) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if p.throttle != nil {
		p.throttle.Take()
	}

	execCtx := ctx
	var cancel context.CancelFunc
	if p.timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	run := func() error {
		if p.cb != nil {
			_, err := p.cb.Execute(func() (any, error) {
				return nil, fn(execCtx)
			})
			return err
		}
		return fn(execCtx)
	}

	if p.retry != nil {
		isRetryable := p.isRetryable
		if isRetryable == nil {
			isRetryable = func(error) bool { return true }
		}
		return Retry(execCtx, *p.retry, isRetryable, run)
	}

	return run()
}
