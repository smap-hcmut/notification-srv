package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

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
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		MinIdleConns: cfg.MinIdleConns,
		PoolSize:     cfg.PoolSize,
		PoolTimeout:  cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}

	// Enable TLS if configured
	if cfg.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis_client.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		Client: client,
		config: cfg,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.Client.Close()
}

// Ping checks if the connection is alive and returns latency
func (c *Client) Ping(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	if err := c.Client.Ping(ctx).Err(); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

// IsConnected checks if the client is connected to Redis
func (c *Client) IsConnected(ctx context.Context) bool {
	_, err := c.Ping(ctx)
	return err == nil
}
