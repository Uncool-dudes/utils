# httpserver

`github.com/uncool-dudes/utils/httpserver`

Chi-based HTTP server. Binds synchronously on start so address-in-use errors surface immediately rather than asynchronously.

## fx

```go
fx.Supply(httpCfg),
logger.Module,
httpserver.Module,
// provides: *httpserver.HttpServer
// OnStart: bind + serve; OnStop: graceful shutdown
```

## Constructor (non-fx)

```go
srv := httpserver.New(cfg, log)
srv.Router().Get("/ping", handler)
err := srv.Start()
```

## HttpServer methods

| Method | Description |
|--------|-------------|
| `srv.Router()` | Return the `*chi.Mux` for mounting routes |
| `srv.Start()` | Bind and serve in the background. Called by fx `OnStart`. |
| `srv.Shutdown(ctx)` | Graceful shutdown. Called by fx `OnStop`. |
| `srv.Reload(cfg)` | Apply timeout config changes at runtime. Addr changes require restart. |

## Config

```go
type Config struct {
    Addr            string        // e.g. ":8080" (required)
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    IdleTimeout     time.Duration
    ShutdownTimeout time.Duration
}
```

## Defaults

| Field | Default |
|-------|---------|
| `ReadTimeout` | `5s` |
| `WriteTimeout` | `10s` |
| `IdleTimeout` | `60s` |
| `ShutdownTimeout` | `10s` |

## Errors

| Sentinel | Meaning |
|----------|---------|
| `httpserver.ErrStartFailed` | Bind failed (e.g. address already in use) |
| `httpserver.ErrShutdown` | Graceful shutdown returned an error |
