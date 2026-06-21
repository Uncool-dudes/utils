package sentry

import (
	"github.com/TheZeroSlave/zapsentry"
	sdk "github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

// NewZapCore returns a zapcore.Core that forwards Error+ log entries to Sentry
// as exceptions, preserving structured fields and stack traces.
// Must be called after Init.
func NewZapCore() (zapcore.Core, error) {
	core, err := zapsentry.NewCore(zapsentry.Configuration{
		Level:         zapcore.ErrorLevel,
		LoggerNameKey: "logger",
	}, zapsentry.NewSentryClientFromClient(sdk.CurrentHub().Client()))
	if err != nil {
		return nil, Domain.Wrap(err, "create sentry zap core")
	}
	return core, nil
}
