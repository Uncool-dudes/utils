package watermill

import (
	"github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
)

type zapLogger struct {
	log    *zap.Logger
	fields watermill.LogFields
}

func newLogger(log *zap.Logger) watermill.LoggerAdapter {
	return &zapLogger{log: log}
}

// piiLogFields are keys Watermill may include in router-level log calls that can
// carry message payload content — strip them to avoid PII in structured logs.
var piiLogFields = map[string]struct{}{
	"payload":  {},
	"metadata": {},
}

func (l *zapLogger) fields2zap(fields watermill.LogFields) []zap.Field {
	all := make(watermill.LogFields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		all[k] = v
	}
	for k, v := range fields {
		all[k] = v
	}
	fs := make([]zap.Field, 0, len(all))
	for k, v := range all {
		if _, skip := piiLogFields[k]; skip {
			continue
		}
		fs = append(fs, zap.Any(k, v))
	}
	return fs
}

func (l *zapLogger) Error(msg string, err error, fields watermill.LogFields) {
	l.log.Error(msg, append(l.fields2zap(fields), zap.Error(err))...)
}

func (l *zapLogger) Info(msg string, fields watermill.LogFields) {
	l.log.Info(msg, l.fields2zap(fields)...)
}

func (l *zapLogger) Debug(msg string, fields watermill.LogFields) {
	l.log.Debug(msg, l.fields2zap(fields)...)
}

func (l *zapLogger) Trace(msg string, fields watermill.LogFields) {
	l.log.Debug(msg, append(l.fields2zap(fields), zap.String("level", "trace"))...)
}

func (l *zapLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	merged := make(watermill.LogFields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return &zapLogger{log: l.log, fields: merged}
}
