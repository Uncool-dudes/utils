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
