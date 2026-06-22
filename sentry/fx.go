package sentry

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap/zapcore"
)

// Module wires Sentry into the fx lifecycle.
// Caller must supply a sentry.Config (via fx.Supply or fx.Provide).
// If DSN is empty, sentry is skipped and a no-op zap core is provided — safe for local dev.
var Module = fx.Module(
	"sentry",
	fx.Provide(fx.Annotate(
		func(cfg Config) (zapcore.Core, error) {
			if cfg.DSN == "" {
				return zapcore.NewNopCore(), nil
			}
			if err := Init(cfg); err != nil {
				return nil, err
			}
			return NewZapCore()
		},
		fx.ResultTags(`group:"logger_cores"`),
	)),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			Flush()
			return nil
		},
	})
}
