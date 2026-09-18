package postgresql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConnector_OpenPingsAndConfiguresPool(t *testing.T) {
	state := registerConnectorTestDriver(t, nil)
	c := NewConnector()

	err := c.Open(context.Background(), DSN{}, Config{
		MaxOpenConnections:      7,
		MaxIdleConnections:      3,
		ConnectionMaxLifetime:   time.Minute,
		ConnectionMaxIdleTime:   time.Second,
		MigrationAdvisoryLockID: 42,
	})
	require.NoError(t, err)
	require.Equal(t, 1, state.pingCalls())
	connector, ok := c.(*connector)
	require.True(t, ok)
	require.Equal(t, 7, connector.db.Stats().MaxOpenConnections)
	require.Equal(t, int64(42), connector.migrationAdvisoryLockID)
	require.NoError(t, c.Close())
	require.Equal(t, 1, state.closeCalls())
}

func TestConnector_OpenUsesTimeoutAndClosesHandleAfterFailedPing(t *testing.T) {
	state := registerConnectorTestDriver(t, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	c := NewConnector()

	err := c.Open(context.Background(), DSN{}, Config{ConnectTimeout: 10 * time.Millisecond})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, state.pingCalls())
	require.Equal(t, 1, state.closeCalls())
	require.Error(t, c.Close())
}

func TestConnector_Ping(t *testing.T) {
	state := registerConnectorTestDriver(t, nil)
	connector := NewConnector()
	require.NoError(t, connector.Open(context.Background(), DSN{}, Config{}))
	t.Cleanup(func() { require.NoError(t, connector.Close()) })

	var pinger Pinger = connector
	require.NoError(t, pinger.Ping(context.Background()))
	require.Equal(t, 2, state.pingCalls())
}

type connectorTestDriver struct {
	state *connectorTestDriverState
}

func (d connectorTestDriver) Open(string) (driver.Conn, error) {
	return &connectorTestConn{state: d.state}, nil
}

type connectorTestDriverState struct {
	mu     sync.Mutex
	ping   func(context.Context) error
	pings  int
	closes int
}

func (s *connectorTestDriverState) pingCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pings
}

func (s *connectorTestDriverState) closeCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closes
}

type connectorTestConn struct {
	state *connectorTestDriverState
}

func (c *connectorTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *connectorTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (c *connectorTestConn) Close() error {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.closes++
	return nil
}
func (c *connectorTestConn) Ping(ctx context.Context) error {
	c.state.mu.Lock()
	c.state.pings++
	ping := c.state.ping
	c.state.mu.Unlock()
	if ping != nil {
		return ping(ctx)
	}
	return nil
}

var connectorTestDriverSequence int

func registerConnectorTestDriver(t *testing.T, ping func(context.Context) error) *connectorTestDriverState {
	t.Helper()
	state := &connectorTestDriverState{ping: ping}
	connectorTestDriverSequence++
	name := "go-kita-postgresql-connector-test-" + strconv.Itoa(connectorTestDriverSequence)
	sql.Register(name, connectorTestDriver{state: state})
	previousDriverName := dbDriverName
	dbDriverName = name
	t.Cleanup(func() { dbDriverName = previousDriverName })
	return state
}
