package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Config defines the minimum Redis connection settings required by the app.
type Config struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Client wraps go-redis with a small app-specific surface.
type Client struct {
	client *goredis.Client
}

func (c *Client) Pipeline() goredis.Pipeliner {
	return c.client.Pipeline()
}

func NewClient(cfg Config) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Client{client: rdb}, nil
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return c.client.Ping(ctx).Err()
}

func (c *Client) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	return c.client.Get(ctx, key).Result()
}

func (c *Client) SAdd(ctx context.Context, key string, members ...string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}

	args := make([]interface{}, len(members))
	for i := range members {
		args[i] = members[i]
	}

	return c.client.SAdd(ctx, key, args...).Result()
}

func (c *Client) SIsMember(ctx context.Context, key string, member string) (bool, error) {
	if c == nil || c.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	return c.client.SIsMember(ctx, key, member).Result()
}
