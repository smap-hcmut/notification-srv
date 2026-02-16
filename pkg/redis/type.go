package redis

import "time"

// Config holds Redis connection configuration
type Config struct {
	Host     string
	Password string
	DB       int
	UseTLS   bool

	// Connection pool settings
	MaxRetries      int
	MinIdleConns    int
	PoolSize        int
	PoolTimeout     time.Duration
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}
