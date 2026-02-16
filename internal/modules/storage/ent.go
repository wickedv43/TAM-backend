package storage

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/logger"

	"github.com/pkg/errors"
	"github.com/samber/do/v2"

	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

// PostgresDB holds database connections and ent client
type PostgresDB struct {
	db    *ent.Client
	sqlDB *sql.DB

	log *zap.SugaredLogger

	rootCtx context.Context
}

func NewPostgres(i do.Injector) (*PostgresDB, error) {
	cfg := do.MustInvoke[*config.Config](i)

	storage, err := do.InvokeStruct[PostgresDB](i)
	if err != nil {
		return nil, errors.Wrap(err, "invoke struct")
	}

	storage.log = do.MustInvoke[*logger.Logger](i).Named("postgres")

	storage.rootCtx, err = do.InvokeNamed[context.Context](i, "root.context")
	if err != nil {
		return nil, errors.Wrap(err, "invoke root.context")
	}

	dsn := cfg.Postgres.DSN
	if !strings.Contains(dsn, "timezone") && !strings.Contains(dsn, "options=") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn = dsn + sep + "options=-c%20timezone%3DUTC"
	}
	storage.sqlDB, err = sql.Open("postgres", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "open sql db")
	}

	storage.sqlDB.SetMaxIdleConns(10)
	storage.sqlDB.SetMaxOpenConns(100)
	storage.sqlDB.SetConnMaxLifetime(30 * time.Minute)
	storage.sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err = storage.sqlDB.PingContext(storage.rootCtx); err != nil {
		return nil, errors.Wrap(err, "ping database")
	}

	storage.log.Info("running database migrations...")
	if err = RunPostgresMigrations(storage.rootCtx, storage.sqlDB); err != nil {
		return nil, errors.Wrap(err, "run migrations")
	}
	storage.log.Info("database migrations completed")

	drv := entsql.OpenDB(dialect.Postgres, storage.sqlDB)
	storage.db = ent.NewClient(ent.Driver(drv))

	return &storage, nil
}

func (p *PostgresDB) Close() error {
	if err := p.db.Close(); err != nil {
		return errors.Wrap(err, "close ent client")
	}
	if err := p.sqlDB.Close(); err != nil {
		return errors.Wrap(err, "close sql db")
	}
	return nil
}
