package redis

import (
	"context"
	"crypto/tls"
	"fmt"

	redis_client "github.com/redis/go-redis/v9"
)

// Client wraps redis.Client with additional functionality
type Client struct {
	*redis_client.Client
	config Config
}

// NewClient creates a new Redis client with the given configuration
func NewClient(cfg Config) (*Client, error) {
	opts := &redis_client.Options{
		Addr:            cfg.Host,
		Password:        cfg.Password,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		MinIdleConns:    cfg.MinIdleConns,
		PoolSize:        cfg.PoolSize,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}

	if cfg.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis_client.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		Client: client,
		config: cfg,
	}, nil
}
