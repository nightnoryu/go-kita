package redis

import (
	"context"
	"errors"
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

// Client is a convenience interface over the subset of Redis commands used by the application.
type Client interface {
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

func NewClient(cfg Config) Client {
	return &client{rdb: goredis.NewClient(&goredis.Options{
		Addr:     cfg.addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})}
}

type client struct {
	rdb *goredis.Client
}

func (c *client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrKeyNotFound
	}
	return value, err
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
