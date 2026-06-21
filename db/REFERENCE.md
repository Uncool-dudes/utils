# db

`github.com/uncool-dudes/utils/db`

`pgxpool` wrapper for PostgreSQL. Connects and pings eagerly on startup; closes the pool on shutdown.

## fx

```go
fx.Supply(dbCfg),
db.Module,
// provides: *db.DBService
// OnStart: connect + ping; OnStop: close pool
```

## Constructor (non-fx)

```go
// For tests and CLIs — connects immediately.
svc, err := db.NewConnected(ctx, cfg)
pool := svc.Pool()
```

## DBService methods

| Method | Description |
|--------|-------------|
| `svc.Pool()` | Return the underlying `*pgxpool.Pool`. Safe to call after `OnStart`. |
| `svc.WithTx(ctx, fn)` | Run `fn` inside a transaction. Rolls back on error, commits on success. |
| `svc.Close()` | Close the pool. Called automatically by fx. |

## Savepoints

```go
// WithSavepoint runs fn inside a savepoint on an existing transaction.
err := db.WithSavepoint(ctx, tx, func(tx pgx.Tx) error {
    // nested operations
    return nil
})
```

## Migrations

```go
// Applies all pending tern migrations from migrationsDir.
// Uses a dedicated connection (not the pool) with postgres advisory locking.
// Safe to call concurrently from multiple replicas.
err := db.Migrate(ctx, connURL, "./migrations")
```

## Config

```go
type Config struct {
    URL                   string        // postgres DSN (required)
    MinConns              int32
    MaxConns              int32
    MaxConnLifetime       time.Duration
    MaxConnLifetimeJitter time.Duration // prevents thundering herd on expiry
    MaxConnIdleTime       time.Duration
    ConnectTimeout        time.Duration
    PingTimeout           time.Duration
    HealthCheckPeriod     time.Duration
}
```

## Defaults

| Field | Default |
|-------|---------|
| `MaxConns` | `20` |
| `MinConns` | `2` |
| `MaxConnLifetime` | `1h` |
| `MaxConnLifetimeJitter` | `5m` |
| `MaxConnIdleTime` | `30m` |
| `ConnectTimeout` | `5s` |
| `PingTimeout` | `5s` |
| `HealthCheckPeriod` | `1m` |

## Errors

| Sentinel | Meaning |
|----------|---------|
| `db.ErrConnFailed` | Pool creation or DSN parse failed |
| `db.ErrPingFailed` | Initial ping after connect failed |
| `db.ErrReloadNotSupported` | Pool config is immutable after init |
