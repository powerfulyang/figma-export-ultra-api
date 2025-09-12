// Package redisx provides Redis client functionality
package redisx

import (
	"context"
	"time"

	"fiber-ent-apollo-pg/internal/config"

	"github.com/redis/go-redis/v9"
)

// Client is an alias for a Redis client
type Client = redis.Client

// Open creates a new Redis client based on configuration
func Open(cfg *config.Config) (*Client, func(), error) {
	if cfg.Redis.Addr == "" {
		return nil, func() {}, nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, func() {}, err
	}
	closer := func() { _ = rdb.Close() }
	return rdb, closer, nil
}
