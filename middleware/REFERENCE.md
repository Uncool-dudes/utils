# middleware

`github.com/uncool-dudes/utils/middleware`

Chi-compatible HTTP middleware: request logging, health endpoints, and structured error responses.

## Request logging

```go
r.Use(middleware.Logger(log))
// Logs per request: method, path, status, bytes written, duration, request_id
```

## Health endpoints

```go
// /healthz — always 200; signals the process is alive
r.Handle("/healthz", middleware.NewLivenessHandler())

// /readyz — 200 when all checks pass, 503 with JSON component detail when any fail
// Results are cached 5s to protect dependencies from probe storms.
r.Handle("/readyz", middleware.NewReadinessHandler(
    middleware.Check{Name: "database", Check: dbSvc.HealthCheck, Timeout: 2 * time.Second},
))
```

## Error handling

```go
// HandlerFunc is like http.HandlerFunc but returns an error.
// Handle adapts it to http.Handler: HTTPError → its status code, all others → 500.
r.Get("/users/{id}", middleware.Handle(func(w http.ResponseWriter, r *http.Request) error {
    user, err := svc.Get(r.Context(), chi.URLParam(r, "id"))
    if errors.Is(err, svc.ErrNotFound) {
        return middleware.NotFound("ERR_USER_NOT_FOUND", "user not found")
    }
    if err != nil {
        return middleware.Internal(err)
    }
    return json.NewEncoder(w).Encode(user)
}))

// WriteError writes a JSON error from a plain http.HandlerFunc.
middleware.WriteError(w, err)
```

### HTTPError constructors

| Constructor | Status |
|-------------|--------|
| `middleware.BadRequest(code, msg)` | 400 |
| `middleware.Unauthorized(code, msg)` | 401 |
| `middleware.NotFound(code, msg)` | 404 |
| `middleware.Unprocessable(code, msg)` | 422 |
| `middleware.Internal(err)` | 500 — code `"ERR_INTERNAL"`, message `"internal server error"` |

JSON response shape:

```json
{"code": "ERR_USER_NOT_FOUND", "message": "user not found"}
```

## Panic recovery

```go
r.Use(middleware.Recovery(log))
// Catches panics, logs with stack trace at ERROR level, returns 500.
// Mount before Logger so panics are still logged with request context.
```

## Distributed rate limiting

Redis-backed, chi-compatible. Fails open on Redis outage.

```go
// IP-based (default)
r.Use(middleware.RateLimit(redisClient, log, middleware.RateLimitConfig{
    Limit:  100,
    Window: time.Minute,
}))

// Per-user via JWT claim already in context
r.Use(middleware.RateLimit(redisClient, log, middleware.RateLimitConfig{
    Limit:     1000,
    Window:    time.Minute,
    KeyMode:   "user",
    UserClaim: "sub",
}))

// Custom key
r.Use(middleware.RateLimit(redisClient, log, middleware.RateLimitConfig{
    KeyMode: "custom",
    KeyFunc: func(r *http.Request) string {
        return r.Header.Get("X-Tenant-ID")
    },
}))
```

Returns `429 Too Many Requests` + `Retry-After` header on limit exceeded.

| Field | Default | Notes |
|---|---|---|
| `Algorithm` | `"sliding_window"` | `"fixed_window"` available |
| `KeyMode` | `"ip"` | `"user"`, `"custom"` |
| `Epsilon` | `0.01` | Sliding window accuracy (1% error) |

## Circuit breaker transport

Wraps `http.Client` outbound calls with a circuit breaker.

```go
cb := resilience.NewCircuitBreaker(resilience.CBDefaults)
client := &http.Client{
    Transport: middleware.CircuitBreakerTransport(cb, http.DefaultTransport),
}
```

See [`resilience/REFERENCE.md`](../resilience/REFERENCE.md) for circuit breaker config.
