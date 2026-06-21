package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
)

// Migrate applies all pending tern migrations from migrationsDir.
// Uses a dedicated pgx connection (not the pool) and postgres advisory locking
// via tern — safe to call concurrently from multiple replicas.
func Migrate(ctx context.Context, connURL, migrationsDir string) error {
	conn, err := pgx.Connect(ctx, connURL)
	if err != nil {
		return Domain.Wrap(err, "migrate: connect")
	}
	defer conn.Close(ctx)

	m, err := migrate.NewMigrator(ctx, conn, "schema_version")
	if err != nil {
		return Domain.Wrap(err, "migrate: new migrator")
	}
	if err := m.LoadMigrations(os.DirFS(migrationsDir)); err != nil {
		return Domain.Wrap(err, "migrate: load")
	}

	return Domain.Wrap(m.Migrate(ctx), "migrate: apply")
}
