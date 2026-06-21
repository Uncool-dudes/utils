package resilience

import (
	"time"

	"go.uber.org/ratelimit"
)

type ThrottleConfig struct {
	// RPS is requests per second. Required.
	RPS int
	// Slack allows up to Slack requests to burst without waiting (default 0 = strict leaky bucket).
	Slack int
}

type Throttle struct {
	rl ratelimit.Limiter
}

func NewThrottle(cfg ThrottleConfig) *Throttle {
	opts := []ratelimit.Option{}
	if cfg.Slack > 0 {
		opts = append(opts, ratelimit.WithSlack(cfg.Slack))
	}
	return &Throttle{rl: ratelimit.New(cfg.RPS, opts...)}
}

// Take blocks until a request slot is available. Returns time waited.
func (t *Throttle) Take() time.Duration {
	start := time.Now()
	t.rl.Take()
	return time.Since(start)
}

// Do throttles then executes fn.
func (t *Throttle) Do(fn func() error) error {
	t.rl.Take()
	return fn()
}
