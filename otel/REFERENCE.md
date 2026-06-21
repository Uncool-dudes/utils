# otel

`github.com/uncool-dudes/utils/otel`

Wires the OpenTelemetry Go SDK: traces, metrics, and logs via OTLP gRPC. Registers global trace, metric, and log providers. Installs W3C TraceContext + Baggage propagation.

## fx

```go
fx.Supply(otelCfg),
otel.Module,
// provides: *otel.Provider
// OnStop: drains spans, flushes metrics and logs (5s timeout)
```

## Constructor (non-fx)

```go
p, err := otel.New(cfg)
defer p.Shutdown(ctx)
```

## Config

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

## Defaults

| Field | Default |
|-------|---------|
| `Endpoint` | `"localhost:4317"` |
| `Insecure` | `true` |
| `SampleRate` | `1.0` |
| `TraceExporter` | `"otlp"` |
| `MetricExporter` | `"otlp"` |
| `LogExporter` | `"otlp"` |

## ExporterKind

| Constant | Value | Notes |
|----------|-------|-------|
| `ExporterOTLP` | `"otlp"` | OTLP gRPC to `Endpoint` |
| `ExporterPrometheus` | `"prometheus"` | Metrics only — exposes a Prometheus scrape endpoint |
| `ExporterStdout` | `"stdout"` | Pretty-print to stdout (dev/debug) |

## HTTP helpers

```go
// Chi middleware: starts a trace span per request, propagates W3C context.
r.Use(otel.Tracing("my-service"))

// Stamp matched chi route onto the active span ("/users/{id}" not "/users/42").
// Wire after Tracing, after chi resolves the route.
r.Use(otel.RouteTag)

// Prometheus /metrics handler — only needed when MetricExporter = "prometheus".
r.Handle("/metrics", otel.MetricsHandler())
```

## Zap bridge

```go
// Forwards every zap entry to the global OTLP log provider.
// Call after otel.New so the global provider is set.
core := otel.ZapCore("my-service")
svc, _ := logger.New(cfg, logger.WithExtraCore(core))
```

## Grafana LGTM (local dev)

```yaml
# docker-compose.yml
services:
  otel-lgtm:
    image: grafana/otel-lgtm
    ports:
      - "4317:4317"  # OTLP gRPC — traces, metrics, logs
      - "3000:3000"  # Grafana UI
```

Point `Endpoint` at `localhost:4317` with `Insecure: true`. All three signals flow into the container; Grafana at `:3000` provides dashboards over Prometheus (metrics), Tempo (traces), and Loki (logs).

## Errors

| Sentinel | Meaning |
|----------|---------|
| `otel.ErrInvalidConfig` | Config failed validation |
| `otel.ErrInit` | Provider initialisation failed |
| `otel.ErrShutdown` | Shutdown returned an error |
