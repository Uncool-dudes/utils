package logger

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides both *Service (for Reload/SetLevel) and *zap.Logger (for injection
// into other packages). Most services only need *zap.Logger.
//
// Caller must supply a logger.Config (via fx.Supply or fx.Provide).
// OnStop flushes buffered entries.
var Module = fx.Module(
	"logger",
	fx.Provide(func(cfg Config) (*Service, error) {
		return New(cfg)
	}),
	fx.Provide(func(svc *Service) *zap.Logger {
		return svc.Logger()
	}),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, svc *Service) {
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return svc.Sync()
		},
	})
}
