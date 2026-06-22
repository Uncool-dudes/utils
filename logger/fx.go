package logger

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Module provides both *Service (for Reload/SetLevel) and *zap.Logger (for injection
// into other packages). Most services only need *zap.Logger.
//
// Caller must supply a logger.Config (via fx.Supply or fx.Provide).
// Extra zapcore.Core values tagged group:"logger_cores" are fanned in automatically
// (e.g. sentry.Module contributes one when DSN is set).
// OnStop flushes buffered entries.
var Module = fx.Module(
	"logger",
	fx.Provide(fx.Annotate(
		func(cfg Config, cores []zapcore.Core) (*Service, error) {
			opts := make([]Option, 0, len(cores))
			for _, c := range cores {
				opts = append(opts, WithExtraCore(c))
			}
			return New(cfg, opts...)
		},
		fx.ParamTags(``, `group:"logger_cores"`),
	)),
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
