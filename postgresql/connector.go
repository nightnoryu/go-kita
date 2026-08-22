package postgresql

import (
	stderrors "errors"
	"io/fs"
	"time"

	"github.com/go-faster/errors"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "postgres" database/sql driver
	"github.com/jmoiron/sqlx"

	"github.com/nightnoryu/go-kita/log"
)

const dbDriverName = "pgx"

type Config struct {
	MaxConnections     int
	ConnectionLifetime time.Duration
	ConnectTimeout     time.Duration
}

func NewConnector() Connector {
	return &connector{}
}

type Connector interface {
	Open(dsn DSN, cfg Config) error
	TransactionalClient() TransactionalClient
	Migrator(logger log.Logger, migrationsFS fs.FS) (Migrator, error)
	Close() error
}

type connector struct {
	db *sqlx.DB
}

func (c *connector) Open(dsn DSN, cfg Config) (err error) {
	c.db, err = sqlx.Open(dbDriverName, dsn.String())
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}
	c.db.SetMaxOpenConns(cfg.MaxConnections)
	c.db.SetConnMaxLifetime(cfg.ConnectionLifetime)
	return nil
}

func (c *connector) TransactionalClient() TransactionalClient {
	return &transactionalClient{c.db}
}

func (c *connector) Migrator(logger log.Logger, migrationsFS fs.FS) (Migrator, error) {
	if c.db == nil {
		return nil, stderrors.New("DB not initialized")
	}
	return &migrator{db: c.db, logger: logger, fs: migrationsFS}, nil
}

func (c *connector) Close() error {
	if c.db != nil {
		err := c.db.Close()
		return errors.Wrap(err, "failed to disconnect")
	}
	return stderrors.New("DB not initialized")
}
