package redis

import (
	"context"
	"fmt"
	"smap-websocket/config"
	pkgRedis "smap-websocket/pkg/redis"
)

var client *pkgRedis.Client

// Connect initializes and returns a Redis client
func Connect(ctx context.Context, cfg config.RedisConfig) (*pkgRedis.Client, error) {
	var err error
	client, err = pkgRedis.NewClient(pkgRedis.Config{
		Host:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.DB,
		UseTLS:          cfg.UseTLS,
		MaxRetries:      cfg.MaxRetries,
		MinIdleConns:    cfg.MinIdleConns,
		PoolSize:        cfg.PoolSize,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
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
