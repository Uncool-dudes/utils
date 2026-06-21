package river

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	riv "github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"go.uber.org/zap"
)

type Client struct {
	client  *riv.Client[pgx.Tx]
	workers *riv.Workers
}

func New(pool *pgxpool.Pool, log *zap.Logger, cfg *Config) (*Client, error) {
	workers := riv.NewWorkers()
	riverCfg := BuildRiverConfig(cfg, workers)

	if cfg != nil {
		for name, q := range cfg.Queues {
			if q.MaxWorkers == 0 {
				log.Info("river queue skipped", zap.String("queue", name))
			}
		}
	}
	for name, q := range riverCfg.Queues {
		log.Info("river queue ready", zap.String("queue", name), zap.Int("workers", q.MaxWorkers))
	}

	client, err := riv.NewClient(riverpgxv5.New(pool), riverCfg)
	if err != nil {
		return nil, Domain.Mark(err, ErrConnect)
	}
	return &Client{client: client, workers: riverCfg.Workers}, nil
}

func (c *Client) Client() *riv.Client[pgx.Tx] { return c.client }
func (c *Client) Workers() *riv.Workers       { return c.workers }

// InsertTx enqueues a job within an existing transaction. The job row commits or
// rolls back atomically with the surrounding business changes — prevents the
// split-brain of "job enqueued but tx rolled back" or "tx committed but enqueue
// failed". Prefer this over Insert for any job triggered by a state change.
//
// Do NOT pass PII in job args — River persists args as JSONB through the full
// job lifecycle and retention period. Pass entity IDs; look up data at execution.
func (c *Client) InsertTx(ctx context.Context, tx pgx.Tx, args riv.JobArgs, opts *riv.InsertOpts) (*rivertype.JobInsertResult, error) {
	result, err := c.client.InsertTx(ctx, tx, args, opts)
	if err != nil {
		return nil, Domain.Wrap(err, "insert job")
	}
	return result, nil
}

// InsertManyTx enqueues multiple jobs atomically within an existing transaction.
//
// Same PII warning as InsertTx — IDs only in args.
func (c *Client) InsertManyTx(ctx context.Context, tx pgx.Tx, params []riv.InsertManyParams) ([]*rivertype.JobInsertResult, error) {
	results, err := c.client.InsertManyTx(ctx, tx, params)
	if err != nil {
		return nil, Domain.Wrap(err, "insert jobs")
	}
	return results, nil
}

// FailedEvents returns a channel of failed-job events and a cancel func.
// Must be called before Client.Start. Fires on every failure (retryable and
// final). Filter by event.Job.State == rivertype.JobStateDiscarded to target
// only jobs that exceeded max attempts — permanently lost work that needs
// alerting.
func (c *Client) FailedEvents() (<-chan *riv.Event, func()) {
	return c.client.Subscribe(riv.EventKindJobFailed)
}
