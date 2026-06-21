package redis

import (
	"context"

	"go.uber.org/fx"
)

// Module provides *Service to the fx container. Caller must supply redis.Config.
var Module = fx.Module(
	"redis",
	fx.Provide(New),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, svc *Service) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error { return svc.connect(ctx) },
		OnStop:  func(_ context.Context) error { return svc.Close() },
	})
}
