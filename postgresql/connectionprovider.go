package postgresql

import (
	"context"
	"sync"
)

type ConnectionProvider interface {
	Connection(ctx context.Context) (TransactionalConnection, error)
}

func NewConnectionProvider(client TransactionalClient) ConnectionProvider {
	return &connectionProvider{
		client:         client,
		connectionPool: map[context.Context]*connectionPoolEntry{},
	}
}

type connectionProvider struct {
	client         TransactionalClient
	mu             sync.Mutex
	connectionPool map[context.Context]*connectionPoolEntry
}

type connectionPoolEntry struct {
	connection *sharedConnection
	count      uint
}

func (provider *connectionProvider) Connection(ctx context.Context) (conn TransactionalConnection, err error) {
	provider.withLock(func() {
		entry, ok := provider.connectionPool[ctx]
		if !ok {
			return
		}

		conn = entry.connection
		entry.count++
	})

	if conn == nil {
		conn, err = provider.client.Connection(ctx)
		if err != nil {
			return conn, err
		}

		sharedConn := &sharedConnection{
			TransactionalConnection: conn,
			ctx:                     ctx,
			releaseCallback:         provider.releaseConnection,
		}

		conn = sharedConn

		provider.withLock(func() {
			provider.connectionPool[ctx] = &connectionPoolEntry{
				connection: sharedConn,
				count:      1,
			}
		})
	}

	return conn, err
}

func (provider *connectionProvider) releaseConnection(ctx context.Context) (err error) {
	provider.withLock(func() {
		entry, ok := provider.connectionPool[ctx]
		if !ok {
			return
		}

		if entry.count == 1 {
			err = entry.connection.close()
			delete(provider.connectionPool, ctx)
			return
		}
		entry.count--
	})
	return
}

func (provider *connectionProvider) withLock(f func()) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	f()
}

type sharedConnection struct {
	TransactionalConnection

	ctx             context.Context
	releaseCallback func(ctx context.Context) error
}

func (conn *sharedConnection) Close() error {
	return conn.releaseCallback(conn.ctx)
}

func (conn *sharedConnection) close() error {
	return conn.TransactionalConnection.Close()
}
