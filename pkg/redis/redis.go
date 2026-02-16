package redis

import (
	"context"
	"time"
)

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
