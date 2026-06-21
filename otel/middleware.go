package otel

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Tracing returns a chi-compatible middleware that starts a trace span per request,
// propagates W3C TraceContext + Baggage from incoming headers, and records
// HTTP semantic attributes (method, route, status code, response size).
//
// Pass the logical service name — it appears as the span operation name prefix.
func Tracing(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(
			next, service,
			otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
		)
	}
}

// MetricsHandler returns an HTTP handler that serves Prometheus metrics from the
// default registry. Mount at /metrics so Prometheus Consul SD can scrape it.
//
//	r.Handle("/metrics", otel.MetricsHandler())
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RouteTag stamps the matched route pattern onto the active OTEL span so traces
// group by route ("/users/{id}") rather than raw URL ("/users/42"). Wire it after
// chi resolves the route so r.Pattern is populated.
//
//	r.Use(otel.Tracing("my-service"))
//	r.Use(otel.RouteTag)
func RouteTag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pattern := r.Pattern; pattern != "" {
			labeler, ok := otelhttp.LabelerFromContext(r.Context())
			if ok {
				labeler.Add(semconv.HTTPRoute(pattern))
			}
		}
		next.ServeHTTP(w, r)
	})
}
