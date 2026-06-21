# logger

`github.com/uncool-dudes/utils/logger`

Zap wrapper with multi-sink fan-out, atomic level control, hot reload, and file rotation via lumberjack.

## fx

```go
fx.Supply(loggerCfg),
logger.Module,
// provides: *logger.Service, *zap.Logger
// OnStop: flushes buffered log entries
```

## Constructor (non-fx)

```go
svc, err := logger.New(cfg, ...opts)
log := svc.Logger()
```

## Service methods

| Method | Description |
|--------|-------------|
| `svc.Logger()` | Return the underlying `*zap.Logger` |
| `svc.Named(name)` | Return a named child logger |
| `svc.SetLevel(lvl)` | Atomically change the root log level at runtime |
| `svc.Reload(cfg, opts...)` | Swap the logger entirely (new sinks, new level) without restart |
| `svc.Sync()` | Flush buffered entries. Suppresses ENOTTY/EINVAL/EBADF from non-file sinks. |

## Options

| Option | Description |
|--------|-------------|
| `WithExtraCore(core)` | Add an extra `zapcore.Core` (e.g. `otel.ZapCore()` for OTLP log bridging, `sentry.NewZapCore()` for Sentry) |
| `WithSamplingHook(fn)` | Called on each sampled/dropped decision |
| `WithPreWriteHook(fn)` | Called before each log entry is written |

## Config

```go
type Config struct {
    Level           string       // root level: debug info warn error dpanic panic fatal
    StacktraceLevel string       // attach stacktrace at and above this level
    Sinks           []SinkConfig // one or more output targets
    SamplingInitial int          // log first N per second; 0 = disabled
    SamplingAfter   int          // log every Nth thereafter
    Development     bool         // enables zap development mode
    DisableCaller   bool
    DisableStack    bool
}
```

### SinkConfig

```go
type SinkConfig struct {
    Path     string       // "stdout", "stderr", or a file path
    Level    string       // optional per-sink level floor; empty = follows root
    Encoding string       // "json" or "console"; defaults to "console" in dev, "json" otherwise
    Rotate   RotateConfig // only applies to file sinks
}
```

### RotateConfig

```go
type RotateConfig struct {
    MaxSizeMB  int  // rotate after N MB
    MaxBackups int  // old files to keep
    MaxAgeDays int  // delete rotated files after N days
    Compress   bool // gzip rotated files
}
```

## Defaults

| Field | Default |
|-------|---------|
| `Level` | `"info"` |
| `StacktraceLevel` | `"error"` |
| `SamplingInitial` | `100` |
| `SamplingAfter` | `100` |
| `Sinks` | `[{Path: "stdout", Encoding: "console"}]` |
| `RotateDefaults.MaxSizeMB` | `10` |
| `RotateDefaults.MaxBackups` | `5` |
| `RotateDefaults.MaxAgeDays` | `30` |
| `RotateDefaults.Compress` | `true` |

## Errors

| Sentinel | Meaning |
|----------|---------|
| `logger.ErrInvalidLevel` | Unrecognised log level string |
| `logger.ErrInvalidEncoding` | Encoding not `"json"` or `"console"` |
