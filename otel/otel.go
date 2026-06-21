// Package otel wires the OpenTelemetry Go SDK: traces, metrics, and logs via
// OTLP gRPC, with Prometheus scrape support for metrics.
//
// Jaeger compatibility: Jaeger v1.35+ accepts OTLP gRPC natively on port 4317.
// Point Endpoint at your Jaeger collector with Insecure=true for local dev.
// W3C TraceContext propagation is used — compatible with Jaeger, Tempo, and
// any OTel-aware proxy (Envoy, nginx-otel, etc.).
package otel

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ratchio/utils/errors"
)

var (
	Domain   = errors.NewDomain("otel")
	validate = validator.New()
)

// Provider holds all three SDK providers and coordinates their shutdown.
type Provider struct {
	tracer *sdktrace.TracerProvider
	meter  *sdkmetric.MeterProvider
	logger *sdklog.LoggerProvider
}

// Shutdown drains buffered spans, flushes metrics, and flushes log records.
// Pass a context with a timeout — the fx module uses 5 seconds.
func (p *Provider) Shutdown(ctx context.Context) error {
	return errors.Combine(
		Domain.Wrapf(p.tracer.Shutdown(ctx), "shutdown tracer provider"),
		errors.Combine(
			Domain.Wrapf(p.meter.Shutdown(ctx), "shutdown meter provider"),
			Domain.Wrapf(p.logger.Shutdown(ctx), "shutdown logger provider"),
		),
	)
}

// New initialises trace, metric, and log providers, registers them as globals,
// and installs W3C TraceContext + Baggage propagation.
// cfg.Disable == true installs noop providers (useful in unit tests).
func New(cfg Config) (*Provider, error) {
	if err := validate.Struct(cfg); err != nil {
		return nil, Domain.Mark(err, ErrInvalidConfig)
	}

	res, err := buildResource(cfg)
	if err != nil {
		return nil, err
	}

	tp, err := buildTracerProvider(cfg, res)
	if err != nil {
		return nil, err
	}

	mp, err := buildMeterProvider(cfg, res)
	if err != nil {
		return nil, err
	}

	lp, err := buildLoggerProvider(cfg, res)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	global.SetLoggerProvider(lp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{tracer: tp, meter: mp, logger: lp}, nil
}

func buildResource(cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{semconv.ServiceName(cfg.ServiceName)}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.ServiceVersion))
	}
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL, attrs...),
	)
	if err != nil {
		return nil, Domain.Wrapf(err, "build otel resource")
	}
	return r, nil
}

func grpcDialOpts(cfg Config) []grpc.DialOption {
	if cfg.Insecure {
		return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	return nil
}

func buildTracerProvider(cfg Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	if cfg.Disable {
		return sdktrace.NewTracerProvider(), nil
	}

	var (
		exp sdktrace.SpanExporter
		err error
	)
	switch cfg.TraceExporter {
	case ExporterStdout:
		exp, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	default: // otlp — compatible with Jaeger 1.35+ on port 4317
		exp, err = otlptracegrpc.New(
			context.Background(),
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
			otlptracegrpc.WithHeaders(cfg.Headers),
			otlptracegrpc.WithTimeout(10*time.Second),
			otlptracegrpc.WithDialOption(grpcDialOpts(cfg)...),
		)
	}
	if err != nil {
		return nil, Domain.Mark(err, ErrInit)
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	), nil
}

func buildMeterProvider(cfg Config, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	if cfg.Disable {
		return sdkmetric.NewMeterProvider(), nil
	}

	var (
		reader sdkmetric.Reader
		err    error
	)
	switch cfg.MetricExporter {
	case ExporterStdout:
		exp, e := stdoutmetric.New()
		if e != nil {
			return nil, Domain.Mark(e, ErrInit)
		}
		reader = sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second))
	case ExporterOTLP:
		exp, e := otlpmetricgrpc.New(
			context.Background(),
			otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
			otlpmetricgrpc.WithHeaders(cfg.Headers),
			otlpmetricgrpc.WithTimeout(10*time.Second),
			otlpmetricgrpc.WithDialOption(grpcDialOpts(cfg)...),
		)
		if e != nil {
			return nil, Domain.Mark(e, ErrInit)
		}
		reader = sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second))
	default: // prometheus — Jaeger doesn't scrape metrics; use a separate Prometheus instance
		reader, err = promexporter.New()
		if err != nil {
			return nil, Domain.Mark(err, ErrInit)
		}
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	), nil
}

func buildLoggerProvider(cfg Config, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	if cfg.Disable {
		return sdklog.NewLoggerProvider(), nil
	}

	var (
		exp sdklog.Exporter
		err error
	)
	switch cfg.LogExporter {
	case ExporterStdout:
		exp, err = stdoutlog.New()
	default: // otlp — route to OTel Collector or a log backend (Loki, etc.)
		exp, err = otlploggrpc.New(
			context.Background(),
			otlploggrpc.WithEndpoint(cfg.Endpoint),
			otlploggrpc.WithHeaders(cfg.Headers),
			otlploggrpc.WithTimeout(10*time.Second),
			otlploggrpc.WithDialOption(grpcDialOpts(cfg)...),
		)
	}
	if err != nil {
		return nil, Domain.Mark(err, ErrInit)
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
		sdklog.WithResource(res),
	), nil
}
