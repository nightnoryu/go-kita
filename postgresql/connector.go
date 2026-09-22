package postgresql

import (
	"context"
	stderrors "errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/go-faster/errors"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "postgres" database/sql driver
	"github.com/jmoiron/sqlx"

	"github.com/nightnoryu/go-kita/log"
)

var dbDriverName = "pgx"

type Config struct {
	// MaxOpenConnections is the maximum number of open connections. Zero keeps
	// database/sql's unlimited default.
	MaxOpenConnections int
	// MaxIdleConnections is the maximum number of idle connections. Zero keeps
	// database/sql's default; use a positive value to configure it explicitly.
	MaxIdleConnections int
	// ConnectionMaxLifetime is the maximum amount of time a connection may be
	// reused. Zero means connections do not expire because of age.
	ConnectionMaxLifetime time.Duration
	// ConnectionMaxIdleTime is the maximum amount of time an idle connection may
	// be retained. Zero means idle connections do not expire because of idleness.
	ConnectionMaxIdleTime time.Duration
	// ConnectTimeout bounds initial connectivity validation. Zero relies only on
	// the deadline or cancellation of the context passed to Open.
	ConnectTimeout time.Duration
	// MigrationAdvisoryLockID identifies this application's migration set. Zero
	// uses Go Kita's legacy default lock ID.
	MigrationAdvisoryLockID int64
}

func NewConnector() Connector {
	return &connector{}
}

type Connector interface {
	// Open configures the pool and verifies connectivity before returning. The
	// connector owns the resulting handle and callers must Close it.
	Open(ctx context.Context, dsn DSN, cfg Config) error
	// Ping verifies database connectivity using ctx. It is the narrow capability
	// applications can use for readiness checks.
	Ping(ctx context.Context) error
	TransactionalClient() TransactionalClient
	Migrator(logger log.Logger, migrationsFS fs.FS) (Migrator, error)
	Close() error
}

type connector struct {
	db                      *sqlx.DB
	migrationAdvisoryLockID int64
}

func (c *connector) Ping(ctx context.Context) error {
	if c.db == nil {
		return stderrors.New("postgresql: database not initialized")
	}
	return c.db.PingContext(ctx)
}

func (c *connector) Open(ctx context.Context, dsn DSN, cfg Config) (err error) {
	if ctx == nil {
		return stderrors.New("postgresql: nil context")
	}
	if c.db != nil {
		return stderrors.New("postgresql: database already initialized")
	}
	if configErr := validateConfig(cfg); configErr != nil {
		return configErr
	}

	db, err := sqlx.Open(dbDriverName, dsn.String())
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}

	configurePool(db, cfg)
	pingCtx := ctx
	var cancel context.CancelFunc
	if cfg.ConnectTimeout > 0 {
		pingCtx, cancel = context.WithTimeout(ctx, cfg.ConnectTimeout)
		defer cancel()
	}
	if err := db.PingContext(pingCtx); err != nil {
		return errors.Join(
			errors.Wrap(err, "failed to verify database connectivity"),
			db.Close(),
		)
	}

	c.db = db
	c.migrationAdvisoryLockID = cfg.MigrationAdvisoryLockID
	if c.migrationAdvisoryLockID == 0 {
		c.migrationAdvisoryLockID = migrationAdvisoryLockID
	}
	return nil
}

func validateConfig(cfg Config) error {
	if cfg.ConnectTimeout < 0 {
		return fmt.Errorf("postgresql: connect timeout must not be negative")
	}
	if cfg.MaxOpenConnections < 0 || cfg.MaxIdleConnections < 0 {
		return fmt.Errorf("postgresql: connection counts must not be negative")
	}
	if cfg.ConnectionMaxLifetime < 0 || cfg.ConnectionMaxIdleTime < 0 {
		return fmt.Errorf("postgresql: connection durations must not be negative")
	}
	return nil
}

func configurePool(db *sqlx.DB, cfg Config) {
	maxOpenConnections := cfg.MaxOpenConnections
	if maxOpenConnections != 0 {
		db.SetMaxOpenConns(maxOpenConnections)
	}
	if cfg.MaxIdleConnections != 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConnections)
	}
	connectionMaxLifetime := cfg.ConnectionMaxLifetime
	if connectionMaxLifetime != 0 {
		db.SetConnMaxLifetime(connectionMaxLifetime)
	}
	if cfg.ConnectionMaxIdleTime != 0 {
		db.SetConnMaxIdleTime(cfg.ConnectionMaxIdleTime)
	}
}

func (c *connector) TransactionalClient() TransactionalClient {
	return &transactionalClient{c.db}
}

func (c *connector) Migrator(logger log.Logger, migrationsFS fs.FS) (Migrator, error) {
	if c.db == nil {
		return nil, stderrors.New("DB not initialized")
	}
	return &migrator{db: c.db, logger: logger, fs: migrationsFS, advisoryLockID: c.migrationAdvisoryLockID}, nil
}

func (c *connector) Close() error {
	if c.db == nil {
		return stderrors.New("DB not initialized")
	}
	db := c.db
	c.db = nil
	if err := db.Close(); err != nil {
		return errors.Wrap(err, "failed to disconnect")
	}
	return nil
}
