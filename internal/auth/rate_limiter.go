package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"smap-websocket/pkg/log"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// MaxConnectionsPerUser is the maximum number of connections per user
	MaxConnectionsPerUser int

	// MaxConnectionsPerUserPerProject is the maximum connections per user per project
	MaxConnectionsPerUserPerProject int

	// MaxConnectionsPerUserPerJob is the maximum connections per user per job
	MaxConnectionsPerUserPerJob int

	// ConnectionRateLimit is the maximum new connections per user per minute
	ConnectionRateLimit int

	// RateLimitWindow is the time window for rate limiting
	RateLimitWindow time.Duration
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxConnectionsPerUser:           10,
		MaxConnectionsPerUserPerProject: 3,
		MaxConnectionsPerUserPerJob:     3,
		ConnectionRateLimit:             20,
		RateLimitWindow:                 time.Minute,
	}
}

// RateLimitError represents a rate limit exceeded error
type RateLimitError struct {
	UserID  string
	Limit   string
	Current int
	Max     int
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for user %s: %s (current: %d, max: %d)", e.UserID, e.Limit, e.Current, e.Max)
}

// IsRateLimitError checks if an error is a RateLimitError
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// ConnectionTracker tracks connection counts and rates
type ConnectionTracker struct {
	// Connection counts per user
	userConnections map[string]int

	// Connection counts per user per project
	userProjectConnections map[string]map[string]int

	// Connection counts per user per job
	userJobConnections map[string]map[string]int

	// Rate limiting: connection timestamps per user
	connectionTimestamps map[string][]time.Time

	mu     sync.RWMutex
	config RateLimitConfig
	logger log.Logger
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

	// Start cleanup goroutine for rate limit timestamps
	go ct.cleanupLoop()

	return ct
}

// CheckAndTrackConnection checks rate limits and tracks a new connection
// Returns an error if any limit is exceeded
func (ct *ConnectionTracker) CheckAndTrackConnection(ctx context.Context, userID, projectID, jobID string) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Check connection rate limit
	if err := ct.checkRateLimitLocked(userID); err != nil {
		ct.logger.Warnf(ctx, "Connection rate limit exceeded for user %s", userID)
		return err
	}

	// Check total connections per user
	currentUserConns := ct.userConnections[userID]
	if currentUserConns >= ct.config.MaxConnectionsPerUser {
		return &RateLimitError{
			UserID:  userID,
			Limit:   "max_connections_per_user",
			Current: currentUserConns,
			Max:     ct.config.MaxConnectionsPerUser,
		}
	}

	// Check project-specific limits
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

	// Check job-specific limits
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

	// All checks passed, track the connection
	ct.trackConnectionLocked(userID, projectID, jobID)

	return nil
}

// checkRateLimitLocked checks the connection rate limit (must be called with lock held)
func (ct *ConnectionTracker) checkRateLimitLocked(userID string) error {
	now := time.Now()
	windowStart := now.Add(-ct.config.RateLimitWindow)

	// Get timestamps within the window
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

	// Add current timestamp
	ct.connectionTimestamps[userID] = append(ct.connectionTimestamps[userID], now)

	return nil
}

// trackConnectionLocked tracks a new connection (must be called with lock held)
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

// UntrackConnection removes tracking for a closed connection
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

// GetUserConnectionCount returns the current connection count for a user
func (ct *ConnectionTracker) GetUserConnectionCount(userID string) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.userConnections[userID]
}

// GetStats returns connection tracking statistics
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

// ConnectionTrackerStats holds connection tracking statistics
type ConnectionTrackerStats struct {
	TotalUsers       int `json:"total_users"`
	TotalConnections int `json:"total_connections"`
}

// cleanupLoop periodically cleans up old rate limit timestamps
func (ct *ConnectionTracker) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ct.cleanupTimestamps()
	}
}

// cleanupTimestamps removes old rate limit timestamps
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
