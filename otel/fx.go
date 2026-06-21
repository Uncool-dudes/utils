package otel

import (
	"context"
	"time"

	"go.uber.org/fx"
)

// Module wires OpenTelemetry into the fx lifecycle.
// Caller must supply an otel.Config (via fx.Supply or fx.Provide).
// OnStart calls New, setting global trace + metric providers.
// OnStop shuts down both providers with a 5-second drain timeout.
var Module = fx.Module(
	"otel",
	fx.Provide(New),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, p *Provider) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return p.Shutdown(ctx)
		},
	})
}
