package river

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	riv "github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module(
	"river",
	fx.Provide(newClient),
)

type clientParams struct {
	fx.In
	Pool   *pgxpool.Pool
	Log    *zap.Logger
	Config Config
}

func newClient(p clientParams) (*Client, error) {
	return New(p.Pool, p.Log, &p.Config)
}

// RegisterWorker adds a job worker to the river client before start.
// Call in an fx.Invoke before river.Module starts.
//
// Heartbeating: River cancels the job context when Config.JobTimeoutSeconds
// elapses — there is no manual heartbeat API. Set JobTimeoutSeconds high enough
// for the slowest expected attempt; too low = spurious rescues and duplicate
// execution of in-flight jobs.
func RegisterWorker[T riv.JobArgs](rc *Client, w riv.Worker[T]) {
	riv.AddWorker(rc.workers, w)
}

// Hooks wires the river client start/stop into the fx lifecycle.
// Include alongside Module in fx.New.
var Hooks = fx.Invoke(registerLifecycle)

func registerLifecycle(lc fx.Lifecycle, rc *Client, log *zap.Logger) {
	// Subscribe BEFORE start — River requires subscriptions registered before client starts.
	failedCh, cancelFailed := rc.FailedEvents()
	// ctx must outlive OnStart — fx cancels the startup ctx after OnStart returns.
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			log.Info("starting river client")
			go func() {
				defer log.Info("river failed watcher stopped")
				for {
					select {
					case event, ok := <-failedCh:
						if !ok {
							return
						}
						// Discarded = exceeded max attempts = permanently lost work. Alert-worthy.
						// Retryable failures are logged at Warn; only discarded at Error.
						if event.Job.State == rivertype.JobStateDiscarded {
							log.Error(
								"river job discarded",
								zap.String("kind", event.Job.Kind),
								zap.Int64("job_id", event.Job.ID),
								zap.Int("attempt", event.Job.Attempt),
							)
						} else {
							log.Warn(
								"river job failed (will retry)",
								zap.String("kind", event.Job.Kind),
								zap.Int64("job_id", event.Job.ID),
								zap.Int("attempt", event.Job.Attempt),
							)
						}
					case <-ctx.Done():
						return
					}
				}
			}()
			return rc.client.Start(startCtx)
		},
		OnStop: func(stopCtx context.Context) error {
			log.Info("stopping river client")
			cancel()
			cancelFailed()
			return rc.client.Stop(stopCtx)
		},
	})
}
