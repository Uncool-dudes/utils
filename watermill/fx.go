package watermill

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides watermill publisher, subscriber, and router to the fx container.
var Module = fx.Module(
	"watermill",
	fx.Provide(newPublisher),
	fx.Provide(newSubscriber),
	fx.Provide(newRouter),
	fx.Invoke(registerLifecycle),
)

type pubParams struct {
	fx.In
	Config Config
	Log    *zap.Logger
}

func newPublisher(p pubParams) (message.Publisher, error) {
	return NewPublisher(p.Config, p.Log)
}

type subParams struct {
	fx.In
	Config Config
	Log    *zap.Logger
}

func newSubscriber(p subParams) (message.Subscriber, error) {
	return NewSubscriber(p.Config, p.Log)
}

type routerParams struct {
	fx.In
	Publisher message.Publisher
	Config    Config
	Log       *zap.Logger
}

func newRouter(p routerParams) (*message.Router, error) {
	return NewRouter(p.Publisher, p.Config, p.Log)
}

func registerLifecycle(lc fx.Lifecycle, router *message.Router, log *zap.Logger) {
	// ctx must outlive the OnStart hook — fx cancels the startup ctx after OnStart returns.
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				defer log.Info("watermill router stopped")
				if err := router.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
					log.Error("watermill router error", zap.Error(err))
				}
			}()
			<-router.Running()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return router.Close()
		},
	})
}
