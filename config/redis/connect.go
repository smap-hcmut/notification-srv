package redis

import (
	"context"
	"fmt"
	"notification-srv/config"

	"github.com/smap-hcmut/shared-libs/go/redis"
)

var client redis.IRedis

// Connect initializes and returns a Redis client
func Connect(ctx context.Context, cfg config.RedisConfig) (redis.IRedis, error) {
	var err error
	client, err = redis.New(redis.RedisConfig{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// Disconnect closes the Redis connection
func Disconnect() error {
	if client != nil {
		return client.Close()
	}
	return nil
}
