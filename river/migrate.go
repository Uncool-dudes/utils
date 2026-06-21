package river

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// Migrate applies all pending River schema migrations (river_jobs, river_queue, etc.).
// Call alongside db.Migrate before starting the River client.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	m, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return Domain.Mark(err, ErrMigrate)
	}
	_, err = m.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return Domain.Wrap(err, "river migrate")
}
