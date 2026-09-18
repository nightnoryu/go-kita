//go:build race

package postgresql

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectionProvider_ConcurrentAcquisitionDoesNotShareConnections(t *testing.T) {
	client := &fakeClient{}
	provider := NewConnectionProvider(client)
	ctx := context.Background()

	const workers = 32
	connections := make(chan TransactionalConnection, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := provider.Connection(ctx)
			if err == nil {
				connections <- conn
			}
		}()
	}
	wg.Wait()
	close(connections)

	for conn := range connections {
		require.NoError(t, conn.Close())
	}
	require.Len(t, client.connections, workers)
	for _, conn := range client.connections {
		require.Equal(t, 1, conn.closeCalls)
	}
}
