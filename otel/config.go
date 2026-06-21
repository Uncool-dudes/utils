package otel

// ExporterKind controls where telemetry data is sent.
type ExporterKind string

// Supported exporter kinds.
const (
	ExporterOTLP       ExporterKind = "otlp"
	ExporterPrometheus ExporterKind = "prometheus" // metrics only
	ExporterStdout     ExporterKind = "stdout"     // dev/debug
)

// Config holds OpenTelemetry SDK initialisation options.
//
// Jaeger (v1.35+) accepts OTLP natively on port 4317 (gRPC) / 4318 (HTTP).
// Set endpoint to your Jaeger collector and insecure=true for local dev.
// The legacy exporters/jaeger exporter is deprecated — use OTLP instead.
type Config struct {
	ServiceName    string `koanf:"service_name"    validate:"required"`
	ServiceVersion string `koanf:"service_version"`
	// Endpoint is the OTLP collector address (host:port, no scheme).
	// For Jaeger: "localhost:4317". For OTel Collector: "otelcol:4317".
	Endpoint       string            `koanf:"endpoint"`
	Headers        map[string]string `koanf:"headers"`  // e.g. auth headers for managed collectors
	Insecure       bool              `koanf:"insecure"` // disable TLS — required for local Jaeger
	SampleRate     float64           `koanf:"sample_rate"      validate:"min=0,max=1"`
	TraceExporter  ExporterKind      `koanf:"trace_exporter"   validate:"omitempty,oneof=otlp stdout"`
	MetricExporter ExporterKind      `koanf:"metric_exporter"  validate:"omitempty,oneof=otlp prometheus stdout"`
	LogExporter    ExporterKind      `koanf:"log_exporter"     validate:"omitempty,oneof=otlp stdout"`
	Disable        bool              `koanf:"disable"` // noop providers — useful in tests
}

// Defaults provides sane out-of-the-box Config values.
//
//nolint:gochecknoglobals // package-level defaults are intentional
var Defaults = Config{
	Endpoint:       "localhost:4317",
	Insecure:       true,
	SampleRate:     1.0,
	TraceExporter:  ExporterOTLP,
	MetricExporter: ExporterOTLP,
	LogExporter:    ExporterOTLP,
	Headers:        map[string]string{},
}
