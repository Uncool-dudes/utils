# resilience

`github.com/uncool-dudes/utils/resilience`

Outbound resilience primitives: circuit breaker, retry with exponential backoff, and outbound rate throttle. Compose them into a `Policy` or use each standalone.

Execution order when all are active: **throttle → timeout → circuit breaker → retry**.

---

## Policy (composed)

```go
policy := resilience.NewPolicy(
    resilience.WithCircuitBreaker(resilience.CBConfig{
        Name:                "payments-api",
        ConsecutiveFailures: 5,
        Timeout:             30 * time.Second,
        OnStateChange: func(name string, from, to gobreaker.State) {
            log.Warn("circuit breaker state change",
                zap.String("name", name),
                zap.String("from", from.String()),
                zap.String("to", to.String()),
            )
        },
    }),
    resilience.WithRetry(resilience.RetryDefaults, func(err error) bool {
        return errors.Is(err, ErrTransient) // only retry transient errors
    }),
    resilience.WithThrottle(resilience.ThrottleConfig{RPS: 50}),
    resilience.WithTimeout(5*time.Second),
)

err := policy.Execute(ctx, func(ctx context.Context) error {
    return callExternalAPI(ctx)
})
```

---

## Circuit breaker

```go
cb := resilience.NewCircuitBreaker(resilience.CBConfig{
    Name:                "stripe",
    ConsecutiveFailures: 5,     // trips after 5 consecutive failures
    Timeout:             60 * time.Second, // stays open 60s before probing
    MaxRequests:         1,     // probe requests in half-open state
})

result, err := cb.Execute(func() (any, error) {
    return stripe.Charge(ctx, params)
})
if errors.Is(err, gobreaker.ErrOpenState) {
    // circuit is open — fail fast
}
```

**States:** closed (normal) → open (failing fast) → half-open (probing) → closed.

Defaults via `resilience.CBDefaults`: 5 consecutive failures, 60s timeout, 1 probe request.

| Field | Default | Notes |
|---|---|---|
| `ConsecutiveFailures` | `5` | Ignored if `ReadyToTrip` is set |
| `Timeout` | `60s` | Open → half-open transition |
| `MaxRequests` | `1` | Probe requests in half-open |
| `Interval` | `0` | Rolling window; 0 = never reset counts |
| `ReadyToTrip` | `nil` | Custom trip logic; overrides `ConsecutiveFailures` |
| `OnStateChange` | `nil` | Wire zap logging here |

---

## Retry

```go
err := resilience.Retry(ctx, resilience.RetryDefaults, func(err error) bool {
    var httpErr *HTTPError
    return errors.As(err, &httpErr) && httpErr.Status >= 500
}, func() error {
    return callAPI()
})
if errors.Is(err, resilience.ErrMaxRetries) {
    // exhausted all attempts
}
```

Defaults via `resilience.RetryDefaults`:

| Field | Default | Notes |
|---|---|---|
| `MaxAttempts` | `3` | Includes first attempt |
| `InitialWait` | `100ms` | |
| `MaxWait` | `10s` | |
| `Multiplier` | `2.0` | Exponential backoff factor |
| `Jitter` | `true` | ±25% spread — prevents thundering herd |

Context cancellation stops retries immediately.

---

## Throttle

Leaky-bucket outbound rate limiter. `Take()` blocks until a slot is available.

```go
throttle := resilience.NewThrottle(resilience.ThrottleConfig{
    RPS:   100,  // max 100 req/s
    Slack: 10,   // allow burst of 10 before blocking
})

// Option 1: manual
waited := throttle.Take()

// Option 2: wrap a call
err := throttle.Do(func() error {
    return callAPI()
})
```

Use for: outbound calls to rate-limited third parties, worker throughput caps.

---

## middleware extensions

See [`middleware/REFERENCE.md`](../middleware/REFERENCE.md) for:
- `middleware.RateLimit` — distributed inbound HTTP rate limiting via Redis
- `middleware.CircuitBreakerTransport` — circuit breaker for outbound `http.Client`

---

## errors

```go
resilience.ErrCircuitOpen  // circuit breaker is open
resilience.ErrMaxRetries   // retry exhausted all attempts
resilience.ErrTimeout      // execution exceeded timeout
```
