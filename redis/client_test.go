package redis

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestClient_Ping(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, listener.Close()) })

	done := make(chan error, 1)
	go func() { done <- servePing(listener) }()

	client := &client{rdb: goredis.NewClient(&goredis.Options{Addr: listener.Addr().String()})}
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Ping(context.Background()))
	require.NoError(t, <-done)
}

func servePing(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("accept Redis client: %w", err)
	}
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	for {
		command, err := readRedisCommand(reader)
		if err != nil {
			return fmt.Errorf("read Redis command: %w", err)
		}
		switch command {
		case "HELLO":
			_, err = io.WriteString(conn, "-ERR unknown command 'HELLO'\r\n")
		case "PING":
			_, err = io.WriteString(conn, "+PONG\r\n")
			if err == nil {
				return nil
			}
		default:
			_, err = io.WriteString(conn, "+OK\r\n")
		}
		if err != nil {
			return fmt.Errorf("write Redis response: %w", err)
		}
	}
}

func readRedisCommand(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(line, "*") {
		return "", fmt.Errorf("expected Redis array, got %q", line)
	}
	count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "*")))
	if err != nil {
		return "", err
	}
	var command string
	for index := 0; index < count; index++ {
		line, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		length, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "$")))
		if err != nil {
			return "", err
		}
		value := make([]byte, length+2)
		if _, err = io.ReadFull(reader, value); err != nil {
			return "", err
		}
		if index == 0 {
			command = strings.ToUpper(string(value[:length]))
		}
	}
	return command, nil
}
