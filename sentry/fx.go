package sentry

import (
	"context"

	"go.uber.org/fx"
)

// Module wires Sentry into the fx lifecycle.
// Caller must supply a sentry.Config (via fx.Supply or fx.Provide).
// OnStart calls Init; OnStop flushes buffered events.
var Module = fx.Module(
	"sentry",
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, cfg Config) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return Init(cfg)
		},
		OnStop: func(_ context.Context) error {
			Flush()
			return nil
		},
	})
}
