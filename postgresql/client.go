package postgresql

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type ClientContext interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

type Transaction interface {
	ClientContext
	Commit() error
	Rollback() error
}

type TransactionalConnection interface {
	ClientContext
	BeginTransaction(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
	Close() error
}

type TransactionalClient interface {
	ClientContext
	BeginTransaction() (Transaction, error)
	Connection(ctx context.Context) (TransactionalConnection, error)
}

type transactionalClient struct {
	*sqlx.DB
}

func (t *transactionalClient) BeginTransaction() (Transaction, error) {
	return t.Beginx()
}

func (t *transactionalClient) Connection(ctx context.Context) (TransactionalConnection, error) {
	connx, err := t.Connx(ctx)
	if err != nil {
		return nil, err
	}
	return &transactionalConnection{Conn: connx}, nil
}

type transactionalConnection struct {
	*sqlx.Conn
}

func (t *transactionalConnection) BeginTransaction(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	return t.BeginTxx(ctx, opts)
}
