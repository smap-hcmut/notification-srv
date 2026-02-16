package auth

import (
	"sync"
	"time"

	"smap-websocket/pkg/log"
)

// AuthorizationError represents an authorization failure
type AuthorizationError struct {
	UserID     string
	ResourceID string
	Resource   string
	Reason     string
}

// RateLimitError represents a rate limit exceeded error
type RateLimitError struct {
	UserID  string
	Limit   string
	Current int
	Max     int
}

// CacheEntry represents a cached authorization result
type CacheEntry struct {
	Allowed   bool
	ExpiresAt time.Time
}

// CachedAuthorizer wraps an Authorizer with caching capabilities
type CachedAuthorizer struct {
	delegate    Authorizer
	cache       map[string]*CacheEntry
	mu          sync.RWMutex
	cacheTTL    time.Duration
	logger      log.Logger
	cacheHits   int64
	cacheMisses int64
}

// PermissiveAuthorizer always allows access (for backward compatibility)
type PermissiveAuthorizer struct {
	logger log.Logger
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	MaxConnectionsPerUser           int
	MaxConnectionsPerUserPerProject int
	MaxConnectionsPerUserPerJob     int
	ConnectionRateLimit             int
	RateLimitWindow                 time.Duration
}

// ConnectionTracker tracks connection counts and rates
type ConnectionTracker struct {
	userConnections        map[string]int
	userProjectConnections map[string]map[string]int
	userJobConnections     map[string]map[string]int
	connectionTimestamps   map[string][]time.Time
	mu                     sync.RWMutex
	config                 RateLimitConfig
	logger                 log.Logger
}

// ConnectionTrackerStats holds connection tracking statistics
type ConnectionTrackerStats struct {
	TotalUsers       int `json:"total_users"`
	TotalConnections int `json:"total_connections"`
}

// SecurityEventType represents the type of security event
type SecurityEventType string

// SecurityEvent represents a security-relevant event
type SecurityEvent struct {
	Type       SecurityEventType      `json:"type"`
	UserID     string                 `json:"user_id"`
	Resource   string                 `json:"resource,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Reason     string                 `json:"reason"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityLogger logs security-relevant events
type SecurityLogger struct {
	logger log.Logger
}
