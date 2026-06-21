package consul

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/ratchio/utils/httpserver"
)

// ModuleFor returns an fx.Module that registers svcName with Consul on startup
// and deregisters on shutdown. Port is derived from httpserver.Config.
func ModuleFor(svcName string) fx.Option {
	return fx.Module(
		"consul",
		fx.Provide(New),
		fx.Invoke(func(lc fx.Lifecycle, c *Client, httpCfg httpserver.Config, log *zap.Logger) {
			_, portStr, err := net.SplitHostPort(httpCfg.Addr)
			if err != nil {
				log.Warn("consul: cannot parse http_server.addr, skipping registration", zap.Error(err))
				return
			}
			port := 8080
			if _, err := fmt.Sscan(portStr, &port); err != nil {
				log.Warn("consul: cannot parse port, skipping registration", zap.Error(err))
				return
			}

			lc.Append(fx.Hook{
				OnStart: func(_ context.Context) error {
					if err := c.Register(svcName, port); err != nil {
						log.Warn("consul: registration failed", zap.Error(err))
					}
					return nil
				},
				OnStop: func(_ context.Context) error {
					if err := c.Deregister(); err != nil {
						log.Warn("consul: deregistration failed", zap.Error(err))
					}
					return nil
				},
			})
		}),
	)
}
