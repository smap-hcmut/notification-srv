package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"smap-websocket/pkg/log"
)

// Authorizer defines the interface for topic access authorization
type Authorizer interface {
	// CanAccessProject checks if a user has access to a specific project
	CanAccessProject(ctx context.Context, userID, projectID string) (bool, error)

	// CanAccessJob checks if a user has access to a specific job
	CanAccessJob(ctx context.Context, userID, jobID string) (bool, error)
}

// AuthorizationError represents an authorization failure
type AuthorizationError struct {
	UserID     string
	ResourceID string
	Resource   string
	Reason     string
}

func (e *AuthorizationError) Error() string {
	return fmt.Sprintf("unauthorized access to %s %s for user %s: %s", e.Resource, e.ResourceID, e.UserID, e.Reason)
}

// IsAuthorizationError checks if an error is an AuthorizationError
func IsAuthorizationError(err error) bool {
	_, ok := err.(*AuthorizationError)
	return ok
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

// NewCachedAuthorizer creates a new CachedAuthorizer
func NewCachedAuthorizer(delegate Authorizer, cacheTTL time.Duration, logger log.Logger) *CachedAuthorizer {
	ca := &CachedAuthorizer{
		delegate: delegate,
		cache:    make(map[string]*CacheEntry),
		cacheTTL: cacheTTL,
		logger:   logger,
	}

	// Start cache cleanup goroutine
	go ca.cleanupLoop()

	return ca
}

// cacheKey generates a cache key for authorization lookups
func cacheKey(userID, resourceType, resourceID string) string {
	return fmt.Sprintf("%s:%s:%s", userID, resourceType, resourceID)
}

// CanAccessProject checks if a user has access to a specific project with caching
func (ca *CachedAuthorizer) CanAccessProject(ctx context.Context, userID, projectID string) (bool, error) {
	key := cacheKey(userID, "project", projectID)

	// Check cache first
	ca.mu.RLock()
	entry, exists := ca.cache[key]
	ca.mu.RUnlock()

	if exists && time.Now().Before(entry.ExpiresAt) {
		ca.cacheHits++
		return entry.Allowed, nil
	}

	ca.cacheMisses++

	// Cache miss or expired, call delegate
	allowed, err := ca.delegate.CanAccessProject(ctx, userID, projectID)
	if err != nil {
		return false, err
	}

	// Update cache
	ca.mu.Lock()
	ca.cache[key] = &CacheEntry{
		Allowed:   allowed,
		ExpiresAt: time.Now().Add(ca.cacheTTL),
	}
	ca.mu.Unlock()

	return allowed, nil
}

// CanAccessJob checks if a user has access to a specific job with caching
func (ca *CachedAuthorizer) CanAccessJob(ctx context.Context, userID, jobID string) (bool, error) {
	key := cacheKey(userID, "job", jobID)

	// Check cache first
	ca.mu.RLock()
	entry, exists := ca.cache[key]
	ca.mu.RUnlock()

	if exists && time.Now().Before(entry.ExpiresAt) {
		ca.cacheHits++
		return entry.Allowed, nil
	}

	ca.cacheMisses++

	// Cache miss or expired, call delegate
	allowed, err := ca.delegate.CanAccessJob(ctx, userID, jobID)
	if err != nil {
		return false, err
	}

	// Update cache
	ca.mu.Lock()
	ca.cache[key] = &CacheEntry{
		Allowed:   allowed,
		ExpiresAt: time.Now().Add(ca.cacheTTL),
	}
	ca.mu.Unlock()

	return allowed, nil
}

// cleanupLoop periodically removes expired cache entries
func (ca *CachedAuthorizer) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ca.cleanup()
	}
}

// cleanup removes expired cache entries
func (ca *CachedAuthorizer) cleanup() {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	now := time.Now()
	for key, entry := range ca.cache {
		if now.After(entry.ExpiresAt) {
			delete(ca.cache, key)
		}
	}
}

// GetCacheStats returns cache statistics
func (ca *CachedAuthorizer) GetCacheStats() (hits, misses int64, size int) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.cacheHits, ca.cacheMisses, len(ca.cache)
}

// InvalidateUser removes all cache entries for a specific user
func (ca *CachedAuthorizer) InvalidateUser(userID string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	prefix := userID + ":"
	for key := range ca.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(ca.cache, key)
		}
	}
}

// InvalidateResource removes cache entries for a specific resource
func (ca *CachedAuthorizer) InvalidateResource(resourceType, resourceID string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	suffix := ":" + resourceType + ":" + resourceID
	for key := range ca.cache {
		if len(key) >= len(suffix) && key[len(key)-len(suffix):] == suffix {
			delete(ca.cache, key)
		}
	}
}
