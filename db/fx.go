package db

import (
	"context"

	"go.uber.org/fx"
)

// Module provides *Service to the fx container.
// Caller must supply a db.Config (via fx.Supply or fx.Provide).
// OnStart connects and pings eagerly; OnStop closes the pool.
var Module = fx.Module(
	"db",
	fx.Provide(New),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, svc *Service) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return svc.connect(ctx)
		},
		OnStop: func(_ context.Context) error {
			svc.Close()
			return nil
		},
	})
}
