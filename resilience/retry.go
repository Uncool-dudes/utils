package resilience

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

type RetryConfig struct {
	// MaxAttempts is total tries including the first (default 3).
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	// Multiplier is the exponential backoff factor (default 2.0).
	Multiplier float64
	// Jitter adds ±25% random spread to each wait to prevent thundering herd.
	Jitter bool
}

var RetryDefaults = RetryConfig{
	MaxAttempts: 3,
	InitialWait: 100 * time.Millisecond,
	MaxWait:     10 * time.Second,
	Multiplier:  2.0,
	Jitter:      true,
}

// Retry executes fn up to cfg.MaxAttempts times. isRetryable decides whether
// an error warrants another attempt. Context cancellation stops retries immediately.
func Retry(ctx context.Context, cfg RetryConfig, isRetryable func(error) bool, fn func() error) error {
	wait := cfg.InitialWait
	var err error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
		if attempt == cfg.MaxAttempts-1 {
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		sleep := wait
		if cfg.Jitter {
			spread := float64(wait) * 0.25
			sleep = wait + time.Duration((rand.Float64()*2-1)*spread)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleep):
		}

		next := time.Duration(math.Round(float64(wait) * cfg.Multiplier))
		if next > cfg.MaxWait {
			next = cfg.MaxWait
		}
		wait = next
	}

	return Domain.Mark(err, ErrMaxRetries)
}
