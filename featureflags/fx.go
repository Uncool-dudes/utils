package featureflags

import (
	"context"

	"go.uber.org/fx"
)

// Module provides *Service to the fx container. Caller must supply openfeature.FeatureProvider.
var Module = fx.Module(
	"featureflags",
	fx.Provide(New),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, svc *Service) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error { return svc.connect(ctx) },
		OnStop:  func(ctx context.Context) error { return svc.Close(ctx) },
	})
}
