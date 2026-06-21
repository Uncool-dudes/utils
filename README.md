# utils

Shared Go infrastructure utilities for uncool-dudes services. Each package is independently importable and fx-compatible. See [REFERENCE.md](REFERENCE.md) for full API, config, and error docs.

```
github.com/uncool-dudes/utils
```

## Packages

| Package | Purpose |
|---------|---------|
| [`errors`](errors/) | Domain-aware error wrapper with stack traces, hints, and Sentry support |
| [`config`](config/) | Generic typed config parser — JSON/YAML/TOML, env overrides, hot reload |
| [`logger`](logger/) | Zap wrapper with multi-sink fan-out, atomic level control, and file rotation |
| [`otel`](otel/) | OpenTelemetry SDK wiring — traces, metrics, and logs via OTLP gRPC |
| [`db`](db/) | pgxpool wrapper — connect, ping, transactions, savepoints, and tern migrations |
| [`httpserver`](httpserver/) | Chi HTTP server with graceful shutdown |
| [`middleware`](middleware/) | Request logging, `/healthz` + `/readyz` handlers, structured error responses |
| [`watermill`](watermill/) | Kafka pub/sub via Watermill — SASL, TLS, retry, poison queue |
| [`river`](river/) | River background job queue backed by PostgreSQL |
| [`consul`](consul/) | Consul service registration and discovery |
| [`pii`](pii/) | PII types (Email, Phone, IP, Name, TaxID) with automatic log redaction |

## Quickstart

A typical service wires these modules together with [uber-go/fx](https://github.com/uber-go/fx):

```go
fx.New(
    // supply configs
    fx.Supply(loggerCfg, otelCfg, dbCfg, httpCfg, watermillCfg, riverCfg, consulCfg),

    // core infrastructure
    logger.Module,
    otel.Module,
    db.Module,
    httpserver.Module,

    // optional: Kafka messaging
    watermill.Module,

    // optional: background jobs
    river.Module,
    river.Hooks,
    fx.Invoke(func(rc *river.Client) {
        river.RegisterWorker(rc, &MyWorker{})
    }),

    // optional: Consul registration
    consul.ModuleFor("my-service"),

    // mount routes
    fx.Invoke(func(srv *httpserver.HttpServer, log *zap.Logger) {
        r := srv.Router()
        r.Use(otel.Tracing("my-service"), otel.RouteTag, middleware.Logger(log))
        r.Handle("/healthz", middleware.NewLivenessHandler())
        r.Handle("/readyz", middleware.NewReadinessHandler())
        r.Get("/users/{id}", middleware.Handle(handleGetUser))
    }),
)
```

## OTLP / Grafana LGTM

Point `otel.Config.Endpoint` at any OTLP gRPC collector. For local development with [grafana/otel-lgtm](https://github.com/grafana/docker-otel-lgtm):

```yaml
# docker-compose.yml
services:
  otel-lgtm:
    image: grafana/otel-lgtm
    ports:
      - "4317:4317"  # OTLP gRPC
      - "3000:3000"  # Grafana
```

```go
cfg := otel.Defaults        // endpoint: localhost:4317, insecure: true
cfg.ServiceName = "my-svc"  // required
```

Wire `otel.ZapCore` into the logger to attach trace/span IDs to every log entry, enabling log→trace correlation in Grafana:

```go
svc, _ := logger.New(loggerCfg, logger.WithExtraCore(otel.ZapCore("my-svc")))
```

## PII safety

Use `pii.*` types for any field that holds personal data. They redact automatically in zap logs and expose a `.Masked()` method for safe display:

```go
log.Info("user created", pii.EmailField("email", user.Email))
// → {"email": "j***@example.com"}
```

Never pass PII in River job args — River persists args as JSONB. Pass entity IDs and look up data at execution time.
