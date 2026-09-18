package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"sync"
)

var (
	// ErrNestedTransaction is returned when a connection already has an active
	// transaction. Nested transactions and concurrent transaction use of one
	// connection are not supported.
	ErrNestedTransaction = errors.New("postgresql: nested transactions are not supported")
	ErrConnectionClosed  = errors.New("postgresql: connection is closed")
)

// ConnectionProvider acquires a new connection for every Connection call.
// The caller owns the returned connection and must close it. Close is
// idempotent, so a deferred Close is safe on all paths.
type ConnectionProvider interface {
	Connection(ctx context.Context) (TransactionalConnection, error)
}

func NewConnectionProvider(client TransactionalClient) ConnectionProvider {
	return &connectionProvider{client: client}
}

type connectionProvider struct {
	client TransactionalClient
}

func (provider *connectionProvider) Connection(ctx context.Context) (TransactionalConnection, error) {
	conn, err := provider.client.Connection(ctx)
	if err != nil {
		return nil, err
	}

	return &managedConnection{TransactionalConnection: conn}, nil
}

// managedConnection makes connection ownership explicit and prevents multiple
// transactions from being started on the same acquired connection.
type managedConnection struct {
	TransactionalConnection

	mu                sync.Mutex
	closed            bool
	transactionActive bool
	closeOnce         sync.Once
	closeErr          error
}

func (conn *managedConnection) BeginTransaction(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.closed {
		return nil, ErrConnectionClosed
	}
	if conn.transactionActive {
		return nil, ErrNestedTransaction
	}

	tx, err := conn.TransactionalConnection.BeginTransaction(ctx, opts)
	if err != nil {
		return nil, err
	}
	conn.transactionActive = true
	return &managedTransaction{Transaction: tx, release: conn.releaseTransaction}, nil
}

func (conn *managedConnection) Close() error {
	conn.closeOnce.Do(func() {
		conn.mu.Lock()
		conn.closed = true
		conn.mu.Unlock()
		conn.closeErr = conn.TransactionalConnection.Close()
	})
	return conn.closeErr
}

func (conn *managedConnection) releaseTransaction() {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	conn.transactionActive = false
}

type managedTransaction struct {
	Transaction

	releaseOnce sync.Once
	release     func()
}

func (tx *managedTransaction) Commit() error {
	err := tx.Transaction.Commit()
	tx.releaseOnce.Do(tx.release)
	return err
}

func (tx *managedTransaction) Rollback() error {
	err := tx.Transaction.Rollback()
	tx.releaseOnce.Do(tx.release)
	return err
}
