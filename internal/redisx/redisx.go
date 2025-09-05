package redisx

import (
	"context"
	"time"

	"fiber-ent-apollo-pg/internal/config"
	"github.com/redis/go-redis/v9"
)

type Client = redis.Client

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
