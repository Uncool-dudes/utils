# sentry

`github.com/uncool-dudes/utils/sentry`

Sentry error tracking. Initialises the global Sentry hub and provides a zap core that forwards `Error+` log entries to Sentry as exceptions.

## fx

```go
fx.Supply(sentryCfg),
sentry.Module,
// OnStart: Init; OnStop: Flush (2s drain)
```

## Constructor (non-fx)

```go
err := sentry.Init(cfg)
defer sentry.Flush()
```

## Functions

| Function | Description |
|----------|-------------|
| `sentry.Init(cfg)` | Initialise the global Sentry hub |
| `sentry.Flush()` | Wait up to 2s for buffered events to be sent. Call before `os.Exit`. |
| `sentry.CaptureException(err)` | Send `err` to Sentry as an exception event with stack trace |
| `sentry.Recover(rv)` | Capture a panic value from `recover()` as a Sentry exception |
| `sentry.RecoverWithContext(ctx, rv)` | Capture a panic with request context so Sentry can attach request metadata |

## Zap bridge

```go
// Forwards Error+ log entries to Sentry as exceptions.
// Must be called after sentry.Init.
core, err := sentry.NewZapCore()
svc, _ := logger.New(cfg, logger.WithExtraCore(core))
```

## Config

```go
type Config struct {
    DSN              string            // Sentry DSN (required)
    Environment      string            // "production" | "staging" | "development"
    ServerName       string
    Tags             map[string]string
    AttachStacktrace bool
    MaxErrorDepth    int
    IgnoreErrors     []string          // error message substrings to suppress
    Debug            bool
    Logging          LoggingOptions
}

type LoggingOptions struct {
    Disable bool // opt out of Sentry log capture (enabled by default since SDK v0.47)
}
```

## Defaults

| Field | Default |
|-------|---------|
| `AttachStacktrace` | `true` |
| `MaxErrorDepth` | `10` |
| `Tags` | `{}` |
| `IgnoreErrors` | `[]` |

## Errors

| Sentinel | Meaning |
|----------|---------|
| `sentry.ErrInvalidConfig` | Config failed validation |
| `sentry.ErrInit` | Sentry SDK initialisation failed |
