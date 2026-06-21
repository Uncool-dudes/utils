package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mennanov/limiters"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RateLimitConfig struct {
	Limit  int64
	Window time.Duration
	// Algorithm is "sliding_window" (default) or "fixed_window".
	// Sliding window is more accurate but uses slightly more Redis storage.
	Algorithm string

	// KeyMode is "ip" (default), "user", or "custom".
	KeyMode string
	// UserClaim is the context key holding the user identifier (e.g. "sub" from JWT).
	// Used when KeyMode is "user". Falls back to IP if claim is missing.
	UserClaim string
	// KeyFunc is called per request when KeyMode is "custom".
	KeyFunc func(r *http.Request) string

	// Epsilon controls sliding window accuracy (default 0.01 = 1% error, lower = more accurate).
	// Only used with sliding_window algorithm.
	Epsilon float64
}

// RateLimit returns chi-compatible distributed rate limiting middleware backed by Redis.
// Returns 429 + Retry-After on limit exceeded. Fails open on Redis errors.
func RateLimit(client *goredis.Client, log *zap.Logger, cfg RateLimitConfig) func(http.Handler) http.Handler {
	if cfg.Algorithm == "" {
		cfg.Algorithm = "sliding_window"
	}
	if cfg.KeyMode == "" {
		cfg.KeyMode = "ip"
	}
	if cfg.Epsilon == 0 {
		cfg.Epsilon = 0.01
	}

	keyFn := resolveKey(cfg)
	clock := limiters.NewSystemClock()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			prefix := "rl:" + key

			var wait time.Duration
			var err error

			switch cfg.Algorithm {
			case "fixed_window":
				inc := limiters.NewFixedWindowRedis(client, prefix)
				lim := limiters.NewFixedWindow(cfg.Limit, cfg.Window, inc, clock)
				wait, err = lim.Limit(r.Context())
			default:
				inc := limiters.NewSlidingWindowRedis(client, prefix)
				lim := limiters.NewSlidingWindow(cfg.Limit, cfg.Window, inc, clock, cfg.Epsilon)
				wait, err = lim.Limit(r.Context())
			}

			if errors.Is(err, limiters.ErrLimitExhausted) {
				log.Warn("rate limit exceeded",
					zap.String("key", key),
					zap.Int64("limit", cfg.Limit),
					zap.Duration("window", cfg.Window),
					zap.String("path", r.URL.Path),
				)
				if wait > 0 {
					w.Header().Set("Retry-After", fmt.Sprintf("%.0f", wait.Seconds()))
				}
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			if err != nil {
				log.Error("rate limiter backend error", zap.Error(err), zap.String("key", key))
				// fail open — don't block traffic on Redis outage
			}

			next.ServeHTTP(w, r)
		})
	}
}

func resolveKey(cfg RateLimitConfig) func(r *http.Request) string {
	switch cfg.KeyMode {
	case "user":
		claim := cfg.UserClaim
		return func(r *http.Request) string {
			if v := r.Context().Value(claim); v != nil {
				if s, ok := v.(string); ok && s != "" {
					return "user:" + s
				}
			}
			return "ip:" + extractIP(r)
		}
	case "custom":
		return cfg.KeyFunc
	default:
		return func(r *http.Request) string {
			return "ip:" + extractIP(r)
		}
	}
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if parts := strings.Split(xff, ","); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[:i]
	}
	return addr
}
