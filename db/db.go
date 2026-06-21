package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/uncool-dudes/utils/errors"
)

var Domain = errors.NewDomain("db")

type DBService struct {
	config Config
	pool   *pgxpool.Pool
}

func New(cfg Config) *DBService {
	return &DBService{config: cfg}
}

// Pool returns the underlying pgxpool. Safe to call after OnStart.
func (o *DBService) Pool() *pgxpool.Pool {
	return o.pool
}

// NewConnected creates a DBService and immediately connects. For use in tests and CLIs.
func NewConnected(ctx context.Context, cfg Config) (*DBService, error) {
	svc := New(cfg)
	if err := svc.connect(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}

func (o *DBService) connect(ctx context.Context) error {
	cfg, err := pgxpool.ParseConfig(o.config.URL)
	if err != nil {
		return Domain.Mark(err, ErrConnFailed)
	}

	if o.config.MinConns > 0 {
		cfg.MinConns = o.config.MinConns
	}
	if o.config.MaxConns > 0 {
		cfg.MaxConns = o.config.MaxConns
	}
	if o.config.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = o.config.MaxConnLifetime
	}
	if o.config.MaxConnLifetimeJitter > 0 {
		cfg.MaxConnLifetimeJitter = o.config.MaxConnLifetimeJitter
	}
	if o.config.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = o.config.MaxConnIdleTime
	}
	if o.config.HealthCheckPeriod > 0 {
		cfg.HealthCheckPeriod = o.config.HealthCheckPeriod
	}
	if o.config.ConnectTimeout > 0 {
		cfg.ConnConfig.ConnectTimeout = o.config.ConnectTimeout
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return Domain.Mark(err, ErrConnFailed)
	}

	pingCtx := ctx
	if o.config.PingTimeout > 0 {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(ctx, o.config.PingTimeout)
		defer cancel()
	}
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return Domain.Mark(err, ErrPingFailed)
	}

	o.pool = pool
	return nil
}

func (o *DBService) Close() {
	if o.pool != nil {
		o.pool.Close()
	}
}

// WithTx runs fn inside a transaction. Rolls back on error, commits on success.
func (o *DBService) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := o.pool.Begin(ctx)
	if err != nil {
		return Domain.Wrap(err, "begin tx")
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return Domain.Wrap(tx.Commit(ctx), "commit tx")
}

// WithSavepoint runs fn inside a savepoint on an existing transaction.
func WithSavepoint(ctx context.Context, tx pgx.Tx, fn func(pgx.Tx) error) error {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return Domain.Wrap(err, "create savepoint")
	}
	if err := fn(sp); err != nil {
		_ = sp.Rollback(ctx)
		return err
	}
	return Domain.Wrap(sp.Commit(ctx), "release savepoint")
}
