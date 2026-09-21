package redis

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestOpenClient_Ping(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, listener.Close()) })

	done := make(chan error, 1)
	go func() { done <- servePing(listener) }()

	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	client, err := OpenClient(context.Background(), DSN{Host: "127.0.0.1", Port: address.Port}, Config{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, <-done)
}

func TestOptions(t *testing.T) {
	tlsConfig := &tls.Config{ServerName: "cache.example.com", MinVersion: tls.VersionTLS12}
	options := options(DSN{Host: "cache", Port: 6379, Password: "secret", DB: 2}, Config{
		DialTimeout:           time.Second,
		ReadTimeout:           2 * time.Second,
		WriteTimeout:          3 * time.Second,
		PoolTimeout:           4 * time.Second,
		MaxActiveConnections:  5,
		ConnectionMaxLifetime: 6 * time.Second,
		TLSConfig:             tlsConfig,
		MaxConnections:        7,
		ConnectionLifetime:    8 * time.Second,
	})

	require.Equal(t, "cache:6379", options.Addr)
	require.Equal(t, "secret", options.Password)
	require.Equal(t, 2, options.DB)
	require.Equal(t, time.Second, options.DialTimeout)
	require.Equal(t, 2*time.Second, options.ReadTimeout)
	require.Equal(t, 3*time.Second, options.WriteTimeout)
	require.Equal(t, 4*time.Second, options.PoolTimeout)
	require.Equal(t, 5, options.MaxActiveConns)
	require.Equal(t, 6*time.Second, options.ConnMaxLifetime)
	require.True(t, options.ContextTimeoutEnabled)
	require.Equal(t, tlsConfig, options.TLSConfig)
	require.NotSame(t, tlsConfig, options.TLSConfig)
}

func TestTranslateGetError(t *testing.T) {
	require.ErrorIs(t, translateGetError(goredis.Nil), ErrKeyNotFound)

	err := errors.New("redis unavailable")
	require.ErrorIs(t, translateGetError(err), err)
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
