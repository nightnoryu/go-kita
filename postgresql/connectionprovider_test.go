package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	mu          sync.Mutex
	connections []*fakeConnection
	connectErr  error
}

func (c *fakeClient) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (c *fakeClient) SelectContext(context.Context, interface{}, string, ...interface{}) error {
	return nil
}
func (c *fakeClient) GetContext(context.Context, interface{}, string, ...interface{}) error {
	return nil
}
func (c *fakeClient) BeginTransaction(context.Context, *sql.TxOptions) (Transaction, error) {
	return nil, nil
}
func (c *fakeClient) Connection(context.Context) (TransactionalConnection, error) {
	if c.connectErr != nil {
		return nil, c.connectErr
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	conn := &fakeConnection{}
	c.connections = append(c.connections, conn)
	return conn, nil
}

type fakeConnection struct {
	mu           sync.Mutex
	closeCalls   int
	closeErr     error
	beginCalls   int
	beginErr     error
	transactions []*fakeTransaction
}

func (c *fakeConnection) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (c *fakeConnection) SelectContext(context.Context, interface{}, string, ...interface{}) error {
	return nil
}
func (c *fakeConnection) GetContext(context.Context, interface{}, string, ...interface{}) error {
	return nil
}
func (c *fakeConnection) BeginTransaction(context.Context, *sql.TxOptions) (Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.beginCalls++
	if c.beginErr != nil {
		return nil, c.beginErr
	}
	tx := &fakeTransaction{}
	c.transactions = append(c.transactions, tx)
	return tx, nil
}
func (c *fakeConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeCalls++
	return c.closeErr
}

type fakeTransaction struct {
	commitErr   error
	rollbackErr error
}

func (tx *fakeTransaction) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (tx *fakeTransaction) SelectContext(context.Context, interface{}, string, ...interface{}) error {
	return nil
}
func (tx *fakeTransaction) GetContext(context.Context, interface{}, string, ...interface{}) error {
	return nil
}
func (tx *fakeTransaction) Commit() error   { return tx.commitErr }
func (tx *fakeTransaction) Rollback() error { return tx.rollbackErr }

func TestConnectionProvider_CloseIsIdempotentAndPreservesCloseFailure(t *testing.T) {
	closeErr := errors.New("close failed")
	client := &fakeClient{}
	provider := NewConnectionProvider(client)
	conn, err := provider.Connection(context.Background())
	require.NoError(t, err)
	client.connections[0].closeErr = closeErr

	require.ErrorIs(t, conn.Close(), closeErr)
	require.ErrorIs(t, conn.Close(), closeErr)
	require.Equal(t, 1, client.connections[0].closeCalls)
	require.ErrorIs(t, func() error {
		_, err := conn.BeginTransaction(context.Background(), nil)
		return err
	}(), ErrConnectionClosed)
}

func TestConnectionProvider_BeginFailureDoesNotLeaveTransactionActive(t *testing.T) {
	beginErr := errors.New("begin failed")
	client := &fakeClient{}
	provider := NewConnectionProvider(client)
	conn, err := provider.Connection(context.Background())
	require.NoError(t, err)
	client.connections[0].beginErr = beginErr

	_, err = conn.BeginTransaction(context.Background(), nil)
	require.ErrorIs(t, err, beginErr)
	client.connections[0].beginErr = nil
	_, err = conn.BeginTransaction(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

func TestConnectionProvider_RejectsNestedTransactionAndReleasesAfterCompletion(t *testing.T) {
	client := &fakeClient{}
	provider := NewConnectionProvider(client)
	conn, err := provider.Connection(context.Background())
	require.NoError(t, err)

	tx, err := conn.BeginTransaction(context.Background(), &sql.TxOptions{ReadOnly: true})
	require.NoError(t, err)
	_, err = conn.BeginTransaction(context.Background(), nil)
	require.ErrorIs(t, err, ErrNestedTransaction)
	require.NoError(t, tx.Rollback())
	_, err = conn.BeginTransaction(context.Background(), nil)
	require.NoError(t, err)
}

func TestConnectionProvider_CommitAndRollbackFailureReleaseTransaction(t *testing.T) {
	client := &fakeClient{}
	provider := NewConnectionProvider(client)
	conn, err := provider.Connection(context.Background())
	require.NoError(t, err)

	commitErr := errors.New("commit failed")
	tx, err := conn.BeginTransaction(context.Background(), nil)
	require.NoError(t, err)
	client.connections[0].transactions[0].commitErr = commitErr
	require.ErrorIs(t, tx.Commit(), commitErr)
	tx, err = conn.BeginTransaction(context.Background(), nil)
	require.NoError(t, err)

	rollbackErr := errors.New("rollback failed")
	client.connections[0].transactions[1].rollbackErr = rollbackErr
	require.ErrorIs(t, tx.Rollback(), rollbackErr)
	_, err = conn.BeginTransaction(context.Background(), nil)
	require.NoError(t, err)
}
