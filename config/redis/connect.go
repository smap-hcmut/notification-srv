package redis

import (
	"context"
	"fmt"
	"notification-srv/config"
	pkgRedis "notification-srv/pkg/redis"
)

var client pkgRedis.IRedis

// Connect initializes and returns a Redis client
func Connect(ctx context.Context, cfg config.RedisConfig) (pkgRedis.IRedis, error) {
	var err error
	client, err = pkgRedis.New(pkgRedis.RedisConfig{
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
