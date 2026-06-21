# Utils Reference

---

## Table of Contents

- [errors](#errors)
- [config](#config)
- [logger](#logger)
- [otel](#otel)
- [db](#db)
- [httpserver](#httpserver)
- [middleware](#middleware)
- [watermill](#watermill)
- [river](#river)
- [consul](#consul)
- [pii](#pii)

---

## errors

`github.com/uncool-dudes/utils/errors`

Domain-aware error wrapper around `cockroachdb/errors`. Errors carry stack traces, domain tags, hints, and safe details that surface in Sentry and structured logs without leaking PII. All other packages declare a `Domain` using this package.

### Domain

```go
var Domain = errors.NewDomain("mypackage")
```

Every package declares one `Domain` at package level. Errors created or wrapped through a domain are tagged with it, allowing `Domain.Has(err)` routing and structured Sentry grouping.

### Constructors

| Method | Description |
|--------|-------------|
| `Domain.New(msg)` | New sentinel error stamped with this domain |
| `Domain.Newf(format, args...)` | Formatted sentinel |
| `Domain.NewCode(code, msg)` | Sentinel with a machine-readable code string |
| `Domain.Wrap(err, msg)` | Wrap an existing error, adding message and stack |
| `Domain.Wrapf(err, format, args...)` | Formatted wrap |
| `Domain.Mark(err, sentinel)` | Stamp `err` so `errors.Is(err, sentinel)` returns true, preserving original message |

### Package-level helpers

| Function | Description |
|----------|-------------|
| `errors.Is(err, target)` | Delegates to `cockroachdb/errors` Is |
| `errors.As(err, target)` | Delegates to `cockroachdb/errors` As |
| `errors.Unwrap(err)` | Unwrap |
| `errors.Combine(err, other)` | Combine two errors; returns the non-nil one if only one is non-nil |
| `errors.WithHint(err, hint)` | Attach a human-readable hint (appears in `%+v` and Sentry) |
| `errors.WithSafeDetail(err, format, args...)` | Attach PII-free telemetry detail |
| `errors.Hints(err)` | Return all attached hints |
| `errors.DomainOf(err)` | Return the domain name of `err`, or empty string |

---

## config

`github.com/uncool-dudes/utils/config`

Generic, typed config parser backed by koanf. Supports JSON, YAML, and TOML files; env var overrides; struct defaults; and hot reload via fsnotify.

### Constructor

```go
cp, err := config.New[AppConfig]("/etc/myapp/config.yaml", ...opts)
cfg := cp.Get()
```

**Errors:** `ErrNotFound` if the file does not exist; `ErrMalformed` if the file cannot be decoded or fails validation.

### Methods

| Method | Description |
|--------|-------------|
| `cp.Get()` | Return the current config. Goroutine-safe. |
| `cp.Watch(func(T, error))` | Register a callback invoked on file changes. Internal state only updates on a successful reload. |

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithEnvPrefix(prefix)` | `"APP"` | Prefix for env var binding. Nested keys use `__` as delimiter: `APP_LOGGER__LEVEL` → `logger.level` |
| `WithDefaults(map[string]any)` | — | Raw koanf key/value defaults |
| `WithDefaultsFrom(v, prefix)` | — | Serialize a struct into koanf defaults. Use `prefix` to scope nested configs (e.g. `"server"`) |
| `WithEnvOverlay(envVarName)` | — | Read `envVarName` at startup (e.g. `APP_ENV=staging`) and load a sibling file `config.staging.yaml` if it exists. Overlay values win over the primary file but lose to env vars. Missing overlay files are silently ignored. |

### Errors

| Sentinel | Meaning |
|----------|---------|
| `config.ErrNotFound` | Config file does not exist |
| `config.ErrMalformed` | File failed to decode or failed struct validation |

---

## logger

`github.com/uncool-dudes/utils/logger`

Zap wrapper with multi-sink fan-out, atomic level control, hot reload, and file rotation via lumberjack.

### fx

```go
fx.Supply(loggerCfg),
logger.Module,
// provides: *logger.Service, *zap.Logger
```

`OnStop` flushes buffered log entries.

### Constructor (non-fx)

```go
svc, err := logger.New(cfg, ...opts)
log := svc.Logger()
```

### Service methods

| Method | Description |
|--------|-------------|
| `svc.Logger()` | Return the underlying `*zap.Logger` |
| `svc.Named(name)` | Return a named child logger |
| `svc.SetLevel(lvl)` | Atomically change the root log level at runtime |
| `svc.Reload(cfg, opts...)` | Swap the logger entirely (new sinks, new level) without restart |
| `svc.Sync()` | Flush buffered entries. Suppresses ENOTTY/EINVAL/EBADF from non-file sinks. |

### Options

| Option | Description |
|--------|-------------|
| `WithExtraCore(core)` | Add an extra `zapcore.Core` (e.g. `otel.ZapCore()` for OTLP log bridging) |
| `WithSamplingHook(fn)` | Called on each sampled/dropped decision |
| `WithPreWriteHook(fn)` | Called before each log entry is written |

### Config

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

#### SinkConfig

```go
type SinkConfig struct {
    Path     string       // "stdout", "stderr", or a file path
    Level    string       // optional per-sink level floor; empty = follows root
    Encoding string       // "json" or "console"; defaults to "console" in dev, "json" otherwise
    Rotate   RotateConfig // only applies to file sinks
}
```

#### RotateConfig

```go
type RotateConfig struct {
    MaxSizeMB  int  // rotate after N MB
    MaxBackups int  // old files to keep
    MaxAgeDays int  // delete rotated files after N days
    Compress   bool // gzip rotated files
}
```

### Defaults

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

### Errors

| Sentinel | Meaning |
|----------|---------|
| `logger.ErrInvalidLevel` | Unrecognised log level string |
| `logger.ErrInvalidEncoding` | Encoding not `"json"` or `"console"` |

---

## otel

`github.com/uncool-dudes/utils/otel`

Wires the OpenTelemetry Go SDK: traces, metrics, and logs via OTLP gRPC, with optional Prometheus scrape support. Registers global trace, metric, and log providers. Installs W3C TraceContext + Baggage propagation.

### fx

```go
fx.Supply(otelCfg),
otel.Module,
// provides: *otel.Provider
// OnStop: drains spans, flushes metrics and logs (5s timeout)
```

### Constructor (non-fx)

```go
p, err := otel.New(cfg)
defer p.Shutdown(ctx)
```

### Config

```go
type Config struct {
    ServiceName    string
    ServiceVersion string
    Endpoint       string            // host:port, no scheme. e.g. "localhost:4317"
    Headers        map[string]string // e.g. auth headers for managed collectors
    Insecure       bool              // disable TLS (required for local dev)
    SampleRate     float64           // 0.0–1.0; >= 1.0 = AlwaysSample
    TraceExporter  ExporterKind      // "otlp" | "stdout"
    MetricExporter ExporterKind      // "otlp" | "prometheus" | "stdout"
    LogExporter    ExporterKind      // "otlp" | "stdout"
    Disable        bool              // install noop providers (useful in tests)
}
```

### Defaults

| Field | Default |
|-------|---------|
| `Endpoint` | `"localhost:4317"` |
| `Insecure` | `true` |
| `SampleRate` | `1.0` |
| `TraceExporter` | `"otlp"` |
| `MetricExporter` | `"otlp"` |
| `LogExporter` | `"otlp"` |

### ExporterKind constants

| Constant | Value | Notes |
|----------|-------|-------|
| `ExporterOTLP` | `"otlp"` | OTLP gRPC to `Endpoint` |
| `ExporterPrometheus` | `"prometheus"` | Metrics only — exposes `/metrics` scrape endpoint |
| `ExporterStdout` | `"stdout"` | Pretty-print to stdout (dev/debug) |

### HTTP helpers

```go
// Chi middleware: starts a trace span per request, propagates W3C context.
r.Use(otel.Tracing("my-service"))

// Stamp matched chi route onto the active span ("/users/{id}" not "/users/42").
// Wire after Tracing and after chi resolves the route.
r.Use(otel.RouteTag)

// Prometheus /metrics handler (only useful when MetricExporter = "prometheus").
r.Handle("/metrics", otel.MetricsHandler())
```

### Zap bridge

```go
// Returns a zapcore.Core that forwards every zap entry to the global OTLP log provider.
// Call after otel.New so the global provider is set.
core := otel.ZapCore("my-service")
svc, _ := logger.New(cfg, logger.WithExtraCore(core))
```

### Errors

| Sentinel | Meaning |
|----------|---------|
| `otel.ErrInvalidConfig` | Config failed validation |
| `otel.ErrInit` | Provider initialisation failed |
| `otel.ErrShutdown` | Shutdown returned an error |

---

## db

`github.com/uncool-dudes/utils/db`

`pgxpool` wrapper for PostgreSQL. Connects and pings on startup; closes the pool on shutdown.

### fx

```go
fx.Supply(dbCfg),
db.Module,
// provides: *db.DBService
// OnStart: connect + ping; OnStop: close pool
```

### Constructor (non-fx)

```go
// For tests and CLIs — connects immediately.
svc, err := db.NewConnected(ctx, cfg)
pool := svc.Pool()
```

### DBService methods

| Method | Description |
|--------|-------------|
| `svc.Pool()` | Return the underlying `*pgxpool.Pool`. Safe to call after `OnStart`. |
| `svc.WithTx(ctx, fn)` | Run `fn` inside a transaction. Rolls back on error, commits on success. |
| `svc.Close()` | Close the pool. Called automatically by fx. |

### Savepoints

```go
// WithSavepoint runs fn inside a savepoint on an existing transaction.
db.WithSavepoint(ctx, tx, func(tx pgx.Tx) error { ... })
```

### Migrations

```go
// Applies all pending tern migrations from migrationsDir.
// Uses a dedicated connection (not the pool) with postgres advisory locking.
// Safe to call from multiple replicas simultaneously.
err := db.Migrate(ctx, connURL, "./migrations")
```

### Config

```go
type Config struct {
    URL                   string        // postgres DSN (required)
    MinConns              int32
    MaxConns              int32
    MaxConnLifetime       time.Duration
    MaxConnLifetimeJitter time.Duration // prevents thundering herd on expiry
    MaxConnIdleTime       time.Duration
    ConnectTimeout        time.Duration
    PingTimeout           time.Duration
    HealthCheckPeriod     time.Duration
}
```

### Defaults

| Field | Default |
|-------|---------|
| `MaxConns` | `20` |
| `MinConns` | `2` |
| `MaxConnLifetime` | `1h` |
| `MaxConnLifetimeJitter` | `5m` |
| `MaxConnIdleTime` | `30m` |
| `ConnectTimeout` | `5s` |
| `PingTimeout` | `5s` |
| `HealthCheckPeriod` | `1m` |

### Errors

| Sentinel | Meaning |
|----------|---------|
| `db.ErrConnFailed` | Pool creation or DSN parse failed |
| `db.ErrPingFailed` | Initial ping after connect failed |
| `db.ErrReloadNotSupported` | Pool config is immutable after init |

---

## httpserver

`github.com/uncool-dudes/utils/httpserver`

Chi-based HTTP server. Binds synchronously on start so address-in-use errors surface immediately rather than asynchronously.

### fx

```go
fx.Supply(httpCfg),
logger.Module,
httpserver.Module,
// provides: *httpserver.HttpServer
// OnStart: bind + serve; OnStop: graceful shutdown
```

### Usage

```go
// Access the chi router to mount routes.
srv.Router().Get("/users/{id}", middleware.Handle(handleGetUser))
```

### HttpServer methods

| Method | Description |
|--------|-------------|
| `srv.Router()` | Return the `*chi.Mux` for mounting routes |
| `srv.Start()` | Bind and serve. Called by fx `OnStart`. |
| `srv.Shutdown(ctx)` | Graceful shutdown. Called by fx `OnStop`. |
| `srv.Reload(cfg)` | Apply timeout config changes at runtime. Addr changes require restart. |

### Config

```go
type Config struct {
    Addr            string        // e.g. ":8080" (required)
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    IdleTimeout     time.Duration
    ShutdownTimeout time.Duration
}
```

### Defaults

| Field | Default |
|-------|---------|
| `ReadTimeout` | `5s` |
| `WriteTimeout` | `10s` |
| `IdleTimeout` | `60s` |
| `ShutdownTimeout` | `10s` |

### Errors

| Sentinel | Meaning |
|----------|---------|
| `httpserver.ErrStartFailed` | Bind failed (e.g. address in use) |
| `httpserver.ErrShutdown` | Shutdown returned an error |

---

## middleware

`github.com/uncool-dudes/utils/middleware`

Chi-compatible HTTP middleware: request logging, health endpoints, and structured error responses.

### Request logging

```go
r.Use(middleware.Logger(log))
// Logs: method, path, status, bytes, duration, request_id
```

### Health endpoints

```go
// /healthz — always 200; signals the process is alive
r.Handle("/healthz", middleware.NewLivenessHandler())

// /readyz — 200 when all checks pass, 503 with JSON detail when any fail
// Results cached 5s to protect dependencies from probe storms.
r.Handle("/readyz", middleware.NewReadinessHandler(
    middleware.Check{Name: "database", Check: dbSvc.HealthCheck, Timeout: 2 * time.Second},
))
```

### Error handling

```go
// HandlerFunc returns an error; Handle adapts it to http.Handler.
r.Get("/users/{id}", middleware.Handle(func(w http.ResponseWriter, r *http.Request) error {
    user, err := svc.Get(r.Context(), chi.URLParam(r, "id"))
    if errors.Is(err, svc.ErrNotFound) {
        return middleware.NotFound("ERR_USER_NOT_FOUND", "user not found")
    }
    if err != nil {
        return middleware.Internal(err)
    }
    json.NewEncoder(w).Encode(user)
    return nil
}))
```

#### HTTPError constructors

| Constructor | Status |
|-------------|--------|
| `middleware.BadRequest(code, msg)` | 400 |
| `middleware.Unauthorized(code, msg)` | 401 |
| `middleware.NotFound(code, msg)` | 404 |
| `middleware.Unprocessable(code, msg)` | 422 |
| `middleware.Internal(err)` | 500 |

`WriteError(w, err)` — write a JSON error directly from a plain `http.HandlerFunc`.

---

## watermill

`github.com/uncool-dudes/utils/watermill`

Kafka publisher/subscriber/router via Watermill. Supports SASL (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512) and mutual TLS.

### fx

```go
fx.Supply(watermillCfg),
logger.Module,
watermill.Module,
// provides: message.Publisher, message.Subscriber, *message.Router
// OnStart: starts router, waits for Running(); OnStop: cancels context + closes router
```

### Publishing

```go
// Publish marshals v as JSON and publishes to topic.
// Sets a correlation ID on the message.
err := watermill.Publish(ctx, pub, "orders.created", myEvent)
```

### Subscribing

```go
// Handle returns a watermill.HandlerFunc that unmarshals the payload into T.
// Unmarshal failures produce a Nack; handler errors propagate to the middleware stack.
router.AddHandler("order-created-handler", "orders.created", sub, "orders.processed", pub,
    watermill.Handle(func(ctx context.Context, evt OrderCreatedEvent) error {
        return processOrder(ctx, evt)
    }),
)
```

### Router middleware stack

Applied automatically by `NewRouter` in this order:

1. **CorrelationID** — propagates trace IDs through message metadata
2. **Logging** — structured per-message latency and outcome via zap
3. **PoisonQueue** — dead-letters permanently failed messages to `cfg.PoisonQueueTopic`
4. **Retry** — exponential backoff up to `cfg.Retry.MaxRetries`
5. **Recoverer** — converts handler panics to errors (feeds into Retry)

### Config

```go
type Config struct {
    Brokers          []string    // required, min 1
    ConsumerGroup    string      // required
    PoisonQueueTopic string      // defaults to "<consumer_group>.failed"
    SASL             SASLConfig
    TLS              TLSConfig
    Retry            RetryConfig
}

type SASLConfig struct {
    Enable    bool
    Mechanism string // "PLAIN" | "SCRAM-SHA-256" | "SCRAM-SHA-512"
    Username  string
    Password  string
}

type TLSConfig struct {
    Enable             bool
    InsecureSkipVerify bool
    CACert             string // path to PEM file
    ClientCert         string // path to PEM file
    ClientKey          string // path to PEM file
}

type RetryConfig struct {
    MaxRetries        int
    InitialIntervalMs int
    Multiplier        float64
}
```

### Defaults

| Field | Default |
|-------|---------|
| `Retry.MaxRetries` | `3` |
| `Retry.InitialIntervalMs` | `100` |
| `Retry.Multiplier` | `2.0` |
| `PoisonQueueTopic` | `"<consumer_group>.failed"` |

### Errors

| Sentinel | Meaning |
|----------|---------|
| `watermill.ErrInvalidConfig` | Config failed validation |
| `watermill.ErrPublisher` | Failed to create Kafka publisher |
| `watermill.ErrSubscriber` | Failed to create Kafka subscriber |
| `watermill.ErrRouter` | Failed to create router |
| `watermill.ErrPublish` | Publish call failed |
| `watermill.ErrMarshal` | JSON marshal of outgoing payload failed |
| `watermill.ErrUnmarshal` | JSON unmarshal of incoming payload failed |

---

## river

`github.com/uncool-dudes/utils/river`

River background job queue backed by PostgreSQL. Workers are registered before the client starts; jobs are enqueued transactionally alongside business writes.

### fx

```go
fx.Supply(riverCfg),
db.Module,
logger.Module,
river.Module,      // provides *river.Client
river.Hooks,       // registers fx lifecycle (start/stop + failed-job watcher)
// Register workers before river.Hooks starts:
fx.Invoke(func(rc *river.Client) {
    river.RegisterWorker(rc, &MyWorker{})
}),
```

`river.Module` and `river.Hooks` are intentionally separate so workers can be registered between them.

### Client methods

| Method | Description |
|--------|-------------|
| `rc.Client()` | Return the underlying `*riv.Client[pgx.Tx]` |
| `rc.Workers()` | Return the `*riv.Workers` registry |
| `rc.InsertTx(ctx, tx, args, opts)` | Enqueue a job within an existing transaction — commits or rolls back atomically with the surrounding business changes |
| `rc.InsertManyTx(ctx, tx, params)` | Enqueue multiple jobs atomically |
| `rc.FailedEvents()` | Return a channel of failed-job events and a cancel func. Must be called before start. |

> **PII warning:** Do not pass PII in job args — River persists args as JSONB through the full job lifecycle and retention period. Pass entity IDs and look up data at execution time.

### Migrations

```go
// Call alongside db.Migrate before starting the River client.
err := river.Migrate(ctx, pool)
```

### Queue helpers

```go
// EnsureQueue guarantees a named queue exists without overwriting existing entries.
river.EnsureQueue(&cfg, "emails", 5)
```

### Config

```go
type Config struct {
    Enabled                     bool
    Queues                      map[string]QueueConfig // queue name → MaxWorkers
    MaxAttempts                 int
    JobTimeoutSeconds           int     // River cancels job context at this deadline
    FetchCooldownMs             int
    FetchPollIntervalMs         int
    RescueStuckJobsAfterMinutes int
    CompletedRetentionHours     int
    DiscardedRetentionHours     int
    CancelledRetentionHours     int
}

type QueueConfig struct {
    MaxWorkers int // required, min 1; set to 0 to disable the queue
}
```

### Defaults

| Field | Default |
|-------|---------|
| `MaxAttempts` | `5` |
| `Queues` | `{"default": {MaxWorkers: 10}}` |
| `CompletedRetentionHours` | `24h` |
| `DiscardedRetentionHours` | `72h` |
| `CancelledRetentionHours` | `24h` |

### Errors

| Sentinel | Meaning |
|----------|---------|
| `river.ErrInvalidConfig` | Config failed validation |
| `river.ErrConnect` | River client creation failed |
| `river.ErrMigrate` | River schema migration failed |

---

## consul

`github.com/uncool-dudes/utils/consul`

Consul service registration and discovery. Registers the service with `/healthz` and `/readyz` checks, and includes the `prometheus` tag and `metrics_path` meta for Prometheus Consul SD.

### fx

```go
fx.Supply(consulCfg),
httpserver.Module,
consul.ModuleFor("my-service"),
// OnStart: Register; OnStop: Deregister
// Registration failures are logged as warnings, not fatal errors.
```

Port is derived automatically from `httpserver.Config.Addr`.

### Client methods (non-fx)

| Method | Description |
|--------|-------------|
| `c.Register(svcName, httpPort)` | Register service with liveness + readiness checks |
| `c.Deregister()` | Remove the registered service |
| `c.Lookup(svcName)` | Return `host:port` of the first healthy instance |

### Config

```go
type Config struct {
    Addr string            // Consul agent address "host:port"
    Tags []string          // additional tags; "prometheus" is always ensured
    Meta map[string]string // included in registration; exposed to Prometheus SD
}
```

### Defaults

| Field | Default |
|-------|---------|
| `Addr` | `"localhost:8500"` |
| `Tags` | `["prometheus"]` |
| `Meta` | `{"metrics_path": "/metrics"}` |

### Errors

| Sentinel | Meaning |
|----------|---------|
| `consul.ErrInvalidAddr` | Addr is not a valid `host:port` |
| `consul.ErrConnect` | Consul client creation failed |
| `consul.ErrRegister` | Service registration failed |
| `consul.ErrDeregister` | Service deregistration failed |
| `consul.ErrLookup` | Health API call failed |
| `consul.ErrNoInstances` | No healthy instances found for the service |

---

## pii

`github.com/uncool-dudes/utils/pii`

Named types for personally identifiable information. All types are thin string (or struct) wrappers that redact automatically when logged via zap, and mask via a `.Masked()` method for safe display.

### Types

| Type | Validate tag | Masked example |
|------|-------------|----------------|
| `pii.Email` | `validate:"email"` | `j***@example.com` |
| `pii.Phone` | `validate:"e164"` | `+91*******210` |
| `pii.IPAddress` | `validate:"ip"` | `192.168.1.xxx` |
| `pii.FirstName` | — | `J***n` |
| `pii.LastName` | — | `D**` |
| `pii.FullName` | — | `J** D**` |
| `pii.TaxID` | — | `AB***67` |

All types implement `zapcore.ObjectMarshaler` — raw values are never emitted to logs.

### Zap field helpers

```go
log.Info("user created",
    pii.EmailField("email", user.Email),
    pii.PhoneField("phone", user.Phone),
    pii.IPField("ip", user.IP),
    pii.FirstNameField("first_name", user.FirstName),
    pii.LastNameField("last_name", user.LastName),
    pii.FullNameField("full_name", user.FullName),
    pii.TaxIDField("tax_id", user.TaxID),
)
// → {"email": "j***@example.com", ...}
```

### IPAddress construction

```go
addr, _ := netip.ParseAddr("192.168.1.42")
ip := pii.NewIPAddress(addr)
ip.Addr()    // netip.Addr
ip.Masked()  // "192.168.1.xxx"
ip.IsValid() // true
```

### Usage in request structs

```go
type CreateUserRequest struct {
    Email pii.Email `validate:"required,email"`
    Phone pii.Phone `validate:"required,e164"`
}
```
