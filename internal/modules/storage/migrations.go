package storage

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
	"github.com/wickedv43/TAM-backend/internal/database/migrations/postgres"
)

// RunPostgresMigrations runs all pending migrations from migrations/postgres folder
func RunPostgresMigrations(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, postgres.Migrations)
	if err != nil {
		return errors.Wrap(err, "failed to initialize postgres migration provider")
	}

	res, err := provider.Up(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to run up postgres migrations")
	}

	for _, r := range res {
		if r.Error != nil {
			return errors.Wrapf(r.Error, "migration %s failed", r.Source.Path)
		}
	}

	return nil
}
