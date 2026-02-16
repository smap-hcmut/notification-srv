package auth

import "time"

const (
	// Default rate limit configuration
	DefaultMaxConnectionsPerUser           = 10
	DefaultMaxConnectionsPerUserPerProject = 3
	DefaultMaxConnectionsPerUserPerJob     = 3
	DefaultConnectionRateLimit             = 20
	DefaultRateLimitWindow                 = time.Minute
)

const (
	SecurityEventAuthorizationFailure SecurityEventType = "authorization_failure"
	SecurityEventRateLimitExceeded    SecurityEventType = "rate_limit_exceeded"
	SecurityEventInvalidInput         SecurityEventType = "invalid_input"
	SecurityEventSuspiciousActivity   SecurityEventType = "suspicious_activity"
)
