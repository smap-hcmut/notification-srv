package auth

import (
	"context"
	"time"

	"smap-websocket/pkg/log"
)

// Authorizer defines the interface for topic access authorization
type Authorizer interface {
	CanAccessProject(ctx context.Context, userID, projectID string) (bool, error)
	CanAccessJob(ctx context.Context, userID, jobID string) (bool, error)
}

// NewCachedAuthorizer creates a new CachedAuthorizer
func NewCachedAuthorizer(delegate Authorizer, cacheTTL time.Duration, logger log.Logger) *CachedAuthorizer {
	ca := &CachedAuthorizer{
		delegate: delegate,
		cache:    make(map[string]*CacheEntry),
		cacheTTL: cacheTTL,
		logger:   logger,
	}
	go ca.cleanupLoop()
	return ca
}

// NewPermissiveAuthorizer creates a new PermissiveAuthorizer
func NewPermissiveAuthorizer(logger log.Logger) *PermissiveAuthorizer {
	return &PermissiveAuthorizer{
		logger: logger,
	}
}

// NewConnectionTracker creates a new ConnectionTracker
func NewConnectionTracker(config RateLimitConfig, logger log.Logger) *ConnectionTracker {
	ct := &ConnectionTracker{
		userConnections:        make(map[string]int),
		userProjectConnections: make(map[string]map[string]int),
		userJobConnections:     make(map[string]map[string]int),
		connectionTimestamps:   make(map[string][]time.Time),
		config:                 config,
		logger:                 logger,
	}
	go ct.cleanupLoop()
	return ct
}

// NewSecurityLogger creates a new SecurityLogger
func NewSecurityLogger(logger log.Logger) *SecurityLogger {
	return &SecurityLogger{
		logger: logger,
	}
}
