// Package redis provides a convenient client for common Redis operations.
package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ErrKeyNotFound is returned when a requested key does not exist.
var ErrKeyNotFound = errors.New("redis: key not found")

// ScoredMember is a member of a sorted set with its score.
type ScoredMember struct {
	Member string
	Score  float64
}

type Config struct {
	// DialTimeout bounds new TCP or TLS connections. Zero uses go-redis's
	// default of five seconds.
	DialTimeout time.Duration
	// ReadTimeout bounds socket reads. Zero uses go-redis's default of five
	// seconds. A context deadline sooner than this timeout takes precedence.
	ReadTimeout time.Duration
	// WriteTimeout bounds socket writes. Zero makes go-redis use ReadTimeout.
	// A context deadline sooner than this timeout takes precedence.
	WriteTimeout time.Duration
	// PoolTimeout bounds waiting for an available pool connection. Zero uses
	// go-redis's default of ReadTimeout plus one second.
	PoolTimeout time.Duration
	// MaxActiveConnections is the maximum number of connections allocated by the
	// pool. Zero leaves the pool unlimited, which is go-redis's default.
	MaxActiveConnections int
	// ConnectionMaxLifetime is the maximum age of a pooled connection. Zero
	// means connections do not expire because of age.
	ConnectionMaxLifetime time.Duration
	// TLSConfig enables TLS when non-nil. It is shallow-cloned when the client is
	// created. Callers must not mutate referenced data after client creation.
	TLSConfig *tls.Config

	// MaxConnections is a deprecated compatibility alias for
	// MaxActiveConnections. MaxActiveConnections takes precedence when non-zero.
	MaxConnections int
	// ConnectionLifetime is a deprecated compatibility alias for
	// ConnectionMaxLifetime. ConnectionMaxLifetime takes precedence when
	// non-zero.
	ConnectionLifetime time.Duration
}

// Client is a convenience interface over the Redis commands demonstrated by Go
// Kita consumers. Applications should define narrower interfaces at their own
// boundaries instead of depending on this interface when they need fewer
// operations.
type Client interface {
	// Ping verifies that Redis is reachable using ctx.
	Ping(ctx context.Context) error
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, keys ...string) error

	ZAdd(ctx context.Context, key string, score float64, member string) error
	ZCard(ctx context.Context, key string) (int64, error)
	ZRangeWithScores(ctx context.Context, key string) ([]ScoredMember, error)
	ZRem(ctx context.Context, key string, members ...string) error

	Close() error
}

func NewClient(dsn DSN, cfg Config) Client {
	return &client{rdb: goredis.NewClient(options(dsn, cfg))}
}

// OpenClient creates a client and verifies Redis connectivity before returning.
// It uses ctx for the validation request and closes the client if validation
// fails. Callers own and must Close a successfully returned client.
func OpenClient(ctx context.Context, dsn DSN, cfg Config) (Client, error) {
	if ctx == nil {
		return nil, errors.New("redis: nil context")
	}

	client := NewClient(dsn, cfg)
	if err := client.Ping(ctx); err != nil {
		return nil, errors.Join(
			fmt.Errorf("redis: verify connectivity: %w", err),
			client.Close(),
		)
	}
	return client, nil
}

func options(dsn DSN, cfg Config) *goredis.Options {
	maxActiveConnections := cfg.MaxActiveConnections
	if maxActiveConnections == 0 {
		maxActiveConnections = cfg.MaxConnections
	}
	connectionMaxLifetime := cfg.ConnectionMaxLifetime
	if connectionMaxLifetime == 0 {
		connectionMaxLifetime = cfg.ConnectionLifetime
	}

	return &goredis.Options{
		Addr:                  dsn.Addr(),
		Password:              dsn.Password,
		DB:                    dsn.DB,
		DialTimeout:           cfg.DialTimeout,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		PoolTimeout:           cfg.PoolTimeout,
		MaxActiveConns:        maxActiveConnections,
		ConnMaxLifetime:       connectionMaxLifetime,
		TLSConfig:             cloneTLSConfig(cfg.TLSConfig),
		ContextTimeoutEnabled: true,
	}
}

func cloneTLSConfig(config *tls.Config) *tls.Config {
	if config == nil {
		return nil
	}
	return config.Clone()
}

type client struct {
	rdb *goredis.Client
}

func (c *client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.rdb.Get(ctx, key).Result()
	return value, translateGetError(err)
}

func translateGetError(err error) error {
	if errors.Is(err, goredis.Nil) {
		return ErrKeyNotFound
	}
	return err
}

func (c *client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.rdb.Expire(ctx, key, ttl).Err()
}

func (c *client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.rdb.Exists(ctx, key).Result()
	return count > 0, err
}

func (c *client) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *client) ZAdd(ctx context.Context, key string, score float64, member string) error {
	return c.rdb.ZAdd(ctx, key, goredis.Z{Score: score, Member: member}).Err()
}

func (c *client) ZCard(ctx context.Context, key string) (int64, error) {
	return c.rdb.ZCard(ctx, key).Result()
}

func (c *client) ZRangeWithScores(ctx context.Context, key string) ([]ScoredMember, error) {
	result, err := c.rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	members := make([]ScoredMember, len(result))
	for i, z := range result {
		members[i] = ScoredMember{Member: z.Member.(string), Score: z.Score} //nolint:forcetypeassert // members are always written as strings
	}
	return members, nil
}

func (c *client) ZRem(ctx context.Context, key string, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	args := make([]any, len(members))
	for i, m := range members {
		args[i] = m
	}
	return c.rdb.ZRem(ctx, key, args...).Err()
}

func (c *client) Close() error {
	return c.rdb.Close()
}
