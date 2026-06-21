package httpserver

import (
	"context"

	"go.uber.org/fx"
)

// Module provides *HttpServer to the fx container.
var Module = fx.Module(
	"httpserver",
	fx.Provide(New),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, svc *HttpServer) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return svc.Start()
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx := ctx
			if svc.config.ShutdownTimeout > 0 {
				var cancel context.CancelFunc
				shutdownCtx, cancel = context.WithTimeout(ctx, svc.config.ShutdownTimeout)
				defer cancel()
			}
			return svc.Shutdown(shutdownCtx)
		},
	})
}
