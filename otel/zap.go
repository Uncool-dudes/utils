package otel

import (
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log/global"
	"go.uber.org/zap/zapcore"
)

// ZapCore returns a zapcore.Core that bridges zap log entries to the global
// OTEL LoggerProvider. Wire it into the logger via logger.WithExtraCore so
// every structured log line is also exported as an OTEL log record — letting
// you correlate logs with traces in your backend (e.g. Grafana Tempo + Loki).
//
// Call after otel.New so the global LoggerProvider is set.
//
//	loggerSvc, _ := logger.New(cfg, logger.WithExtraCore(otel.ZapCore("my-service")))
func ZapCore(service string) zapcore.Core {
	return otelzap.NewCore(service, otelzap.WithLoggerProvider(global.GetLoggerProvider()))
}
