package auth

import (
	"context"
	"fmt"
	"time"
)

// Error implementations
func (e *AuthorizationError) Error() string {
	return fmt.Sprintf("unauthorized access to %s %s for user %s: %s", e.Resource, e.ResourceID, e.UserID, e.Reason)
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for user %s: %s (current: %d, max: %d)", e.UserID, e.Limit, e.Current, e.Max)
}

// IsAuthorizationError checks if an error is an AuthorizationError
func IsAuthorizationError(err error) bool {
	_, ok := err.(*AuthorizationError)
	return ok
}

// IsRateLimitError checks if an error is a RateLimitError
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxConnectionsPerUser:           DefaultMaxConnectionsPerUser,
		MaxConnectionsPerUserPerProject: DefaultMaxConnectionsPerUserPerProject,
		MaxConnectionsPerUserPerJob:     DefaultMaxConnectionsPerUserPerJob,
		ConnectionRateLimit:             DefaultConnectionRateLimit,
		RateLimitWindow:                 DefaultRateLimitWindow,
	}
}

// CachedAuthorizer methods
func cacheKey(userID, resourceType, resourceID string) string {
	return fmt.Sprintf("%s:%s:%s", userID, resourceType, resourceID)
}

func (ca *CachedAuthorizer) CanAccessProject(ctx context.Context, userID, projectID string) (bool, error) {
	key := cacheKey(userID, "project", projectID)

	ca.mu.RLock()
	entry, exists := ca.cache[key]
	ca.mu.RUnlock()

	if exists && time.Now().Before(entry.ExpiresAt) {
		ca.cacheHits++
		return entry.Allowed, nil
	}

	ca.cacheMisses++

	allowed, err := ca.delegate.CanAccessProject(ctx, userID, projectID)
	if err != nil {
		return false, err
	}

	ca.mu.Lock()
	ca.cache[key] = &CacheEntry{
		Allowed:   allowed,
		ExpiresAt: time.Now().Add(ca.cacheTTL),
	}
	ca.mu.Unlock()

	return allowed, nil
}

func (ca *CachedAuthorizer) CanAccessJob(ctx context.Context, userID, jobID string) (bool, error) {
	key := cacheKey(userID, "job", jobID)

	ca.mu.RLock()
	entry, exists := ca.cache[key]
	ca.mu.RUnlock()

	if exists && time.Now().Before(entry.ExpiresAt) {
		ca.cacheHits++
		return entry.Allowed, nil
	}

	ca.cacheMisses++

	allowed, err := ca.delegate.CanAccessJob(ctx, userID, jobID)
	if err != nil {
		return false, err
	}

	ca.mu.Lock()
	ca.cache[key] = &CacheEntry{
		Allowed:   allowed,
		ExpiresAt: time.Now().Add(ca.cacheTTL),
	}
	ca.mu.Unlock()

	return allowed, nil
}

func (ca *CachedAuthorizer) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ca.cleanup()
	}
}

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

func (ca *CachedAuthorizer) GetCacheStats() (hits, misses int64, size int) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.cacheHits, ca.cacheMisses, len(ca.cache)
}

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

// PermissiveAuthorizer methods
func (pa *PermissiveAuthorizer) CanAccessProject(ctx context.Context, userID, projectID string) (bool, error) {
	pa.logger.Debugf(ctx, "Permissive authorization: allowing user %s access to project %s", userID, projectID)
	return true, nil
}

func (pa *PermissiveAuthorizer) CanAccessJob(ctx context.Context, userID, jobID string) (bool, error) {
	pa.logger.Debugf(ctx, "Permissive authorization: allowing user %s access to job %s", userID, jobID)
	return true, nil
}

// ConnectionTracker methods
func (ct *ConnectionTracker) CheckAndTrackConnection(ctx context.Context, userID, projectID, jobID string) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if err := ct.checkRateLimitLocked(userID); err != nil {
		ct.logger.Warnf(ctx, "Connection rate limit exceeded for user %s", userID)
		return err
	}

	currentUserConns := ct.userConnections[userID]
	if currentUserConns >= ct.config.MaxConnectionsPerUser {
		return &RateLimitError{
			UserID:  userID,
			Limit:   "max_connections_per_user",
			Current: currentUserConns,
			Max:     ct.config.MaxConnectionsPerUser,
		}
	}

	if projectID != "" {
		if ct.userProjectConnections[userID] == nil {
			ct.userProjectConnections[userID] = make(map[string]int)
		}
		currentProjectConns := ct.userProjectConnections[userID][projectID]
		if currentProjectConns >= ct.config.MaxConnectionsPerUserPerProject {
			return &RateLimitError{
				UserID:  userID,
				Limit:   "max_connections_per_user_per_project",
				Current: currentProjectConns,
				Max:     ct.config.MaxConnectionsPerUserPerProject,
			}
		}
	}

	if jobID != "" {
		if ct.userJobConnections[userID] == nil {
			ct.userJobConnections[userID] = make(map[string]int)
		}
		currentJobConns := ct.userJobConnections[userID][jobID]
		if currentJobConns >= ct.config.MaxConnectionsPerUserPerJob {
			return &RateLimitError{
				UserID:  userID,
				Limit:   "max_connections_per_user_per_job",
				Current: currentJobConns,
				Max:     ct.config.MaxConnectionsPerUserPerJob,
			}
		}
	}

	ct.trackConnectionLocked(userID, projectID, jobID)
	return nil
}

func (ct *ConnectionTracker) checkRateLimitLocked(userID string) error {
	now := time.Now()
	windowStart := now.Add(-ct.config.RateLimitWindow)

	timestamps := ct.connectionTimestamps[userID]
	validTimestamps := make([]time.Time, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts.After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	ct.connectionTimestamps[userID] = validTimestamps

	if len(validTimestamps) >= ct.config.ConnectionRateLimit {
		return &RateLimitError{
			UserID:  userID,
			Limit:   "connection_rate_limit",
			Current: len(validTimestamps),
			Max:     ct.config.ConnectionRateLimit,
		}
	}

	ct.connectionTimestamps[userID] = append(ct.connectionTimestamps[userID], now)
	return nil
}

func (ct *ConnectionTracker) trackConnectionLocked(userID, projectID, jobID string) {
	ct.userConnections[userID]++

	if projectID != "" {
		if ct.userProjectConnections[userID] == nil {
			ct.userProjectConnections[userID] = make(map[string]int)
		}
		ct.userProjectConnections[userID][projectID]++
	}

	if jobID != "" {
		if ct.userJobConnections[userID] == nil {
			ct.userJobConnections[userID] = make(map[string]int)
		}
		ct.userJobConnections[userID][jobID]++
	}
}

func (ct *ConnectionTracker) UntrackConnection(userID, projectID, jobID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.userConnections[userID] > 0 {
		ct.userConnections[userID]--
		if ct.userConnections[userID] == 0 {
			delete(ct.userConnections, userID)
		}
	}

	if projectID != "" && ct.userProjectConnections[userID] != nil {
		if ct.userProjectConnections[userID][projectID] > 0 {
			ct.userProjectConnections[userID][projectID]--
			if ct.userProjectConnections[userID][projectID] == 0 {
				delete(ct.userProjectConnections[userID], projectID)
			}
		}
		if len(ct.userProjectConnections[userID]) == 0 {
			delete(ct.userProjectConnections, userID)
		}
	}

	if jobID != "" && ct.userJobConnections[userID] != nil {
		if ct.userJobConnections[userID][jobID] > 0 {
			ct.userJobConnections[userID][jobID]--
			if ct.userJobConnections[userID][jobID] == 0 {
				delete(ct.userJobConnections[userID], jobID)
			}
		}
		if len(ct.userJobConnections[userID]) == 0 {
			delete(ct.userJobConnections, userID)
		}
	}
}

func (ct *ConnectionTracker) GetUserConnectionCount(userID string) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.userConnections[userID]
}

func (ct *ConnectionTracker) GetStats() ConnectionTrackerStats {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	totalUsers := len(ct.userConnections)
	totalConnections := 0
	for _, count := range ct.userConnections {
		totalConnections += count
	}

	return ConnectionTrackerStats{
		TotalUsers:       totalUsers,
		TotalConnections: totalConnections,
	}
}

func (ct *ConnectionTracker) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ct.cleanupTimestamps()
	}
}

func (ct *ConnectionTracker) cleanupTimestamps() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	windowStart := time.Now().Add(-ct.config.RateLimitWindow)

	for userID, timestamps := range ct.connectionTimestamps {
		validTimestamps := make([]time.Time, 0, len(timestamps))
		for _, ts := range timestamps {
			if ts.After(windowStart) {
				validTimestamps = append(validTimestamps, ts)
			}
		}
		if len(validTimestamps) == 0 {
			delete(ct.connectionTimestamps, userID)
		} else {
			ct.connectionTimestamps[userID] = validTimestamps
		}
	}
}

// SecurityLogger methods
func (sl *SecurityLogger) LogAuthorizationFailure(ctx context.Context, userID, resource, resourceID, reason string) {
	sl.logger.Warnf(ctx, "SECURITY: Authorization failure - user=%s resource=%s resourceID=%s reason=%s",
		userID, resource, resourceID, reason)
}

func (sl *SecurityLogger) LogRateLimitExceeded(ctx context.Context, userID, limitType string, current, max int) {
	sl.logger.Warnf(ctx, "SECURITY: Rate limit exceeded - user=%s limit=%s current=%d max=%d",
		userID, limitType, current, max)
}

func (sl *SecurityLogger) LogInvalidInput(ctx context.Context, userID, field, value, reason string) {
	sl.logger.Warnf(ctx, "SECURITY: Invalid input - user=%s field=%s reason=%s",
		userID, field, reason)
}

func (sl *SecurityLogger) LogSuspiciousActivity(ctx context.Context, userID, activity string, metadata map[string]interface{}) {
	sl.logger.Warnf(ctx, "SECURITY: Suspicious activity - user=%s activity=%s",
		userID, activity)
}
