# river

`github.com/uncool-dudes/utils/river`

River background job queue backed by PostgreSQL. Workers are registered before the client starts; jobs are enqueued transactionally alongside business writes to prevent split-brain between enqueue and commit.

## fx

```go
fx.Supply(riverCfg),
db.Module,
logger.Module,
river.Module,   // provides *river.Client
river.Hooks,    // registers fx lifecycle (start/stop + failed-job watcher)

// Register workers before Hooks starts:
fx.Invoke(func(rc *river.Client) {
    river.RegisterWorker(rc, &MyWorker{})
}),
```

`river.Module` and `river.Hooks` are intentionally separate so workers can be registered between them.

## Client methods

| Method | Description |
|--------|-------------|
| `rc.Client()` | Return the underlying `*riv.Client[pgx.Tx]` |
| `rc.Workers()` | Return the `*riv.Workers` registry |
| `rc.InsertTx(ctx, tx, args, opts)` | Enqueue a job within an existing transaction — commits or rolls back atomically with surrounding business changes |
| `rc.InsertManyTx(ctx, tx, params)` | Enqueue multiple jobs atomically |
| `rc.FailedEvents()` | Return a `(<-chan *riv.Event, cancelFunc)`. Must be called before start. Fires on every failure; filter by `JobStateDiscarded` for permanently lost work. |

> **PII warning:** Do not pass PII in job args — River persists args as JSONB through the full job lifecycle and retention period. Pass entity IDs and look up data at execution time.

## Migrations

```go
// Call alongside db.Migrate before starting the River client.
err := river.Migrate(ctx, pool)
```

## Queue helpers

```go
// EnsureQueue guarantees a named queue exists without overwriting existing entries.
river.EnsureQueue(&cfg, "emails", 5)
```

## Config

```go
type Config struct {
    Enabled                     bool
    Queues                      map[string]QueueConfig // queue name → MaxWorkers; set MaxWorkers=0 to disable
    MaxAttempts                 int
    JobTimeoutSeconds           int     // River cancels the job context at this deadline
    FetchCooldownMs             int
    FetchPollIntervalMs         int
    RescueStuckJobsAfterMinutes int
    CompletedRetentionHours     int
    DiscardedRetentionHours     int
    CancelledRetentionHours     int
}
```

## Defaults

| Field | Default |
|-------|---------|
| `MaxAttempts` | `5` |
| `Queues` | `{"default": {MaxWorkers: 10}}` |
| `CompletedRetentionHours` | `24h` |
| `DiscardedRetentionHours` | `72h` |
| `CancelledRetentionHours` | `24h` |

## Errors

| Sentinel | Meaning |
|----------|---------|
| `river.ErrInvalidConfig` | Config failed validation |
| `river.ErrConnect` | River client creation failed |
| `river.ErrMigrate` | River schema migration failed |
