package auth

import (
	"context"
	"time"

	"smap-websocket/pkg/log"
)

// SecurityEventType represents the type of security event
type SecurityEventType string

const (
	SecurityEventAuthorizationFailure SecurityEventType = "authorization_failure"
	SecurityEventRateLimitExceeded    SecurityEventType = "rate_limit_exceeded"
	SecurityEventInvalidInput         SecurityEventType = "invalid_input"
	SecurityEventSuspiciousActivity   SecurityEventType = "suspicious_activity"
)

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

// NewSecurityLogger creates a new SecurityLogger
func NewSecurityLogger(logger log.Logger) *SecurityLogger {
	return &SecurityLogger{
		logger: logger,
	}
}

// LogAuthorizationFailure logs an authorization failure event
func (sl *SecurityLogger) LogAuthorizationFailure(ctx context.Context, userID, resource, resourceID, reason string) {
	event := SecurityEvent{
		Type:       SecurityEventAuthorizationFailure,
		UserID:     userID,
		Resource:   resource,
		ResourceID: resourceID,
		Reason:     reason,
		Timestamp:  time.Now(),
	}
	sl.logger.Warnf(ctx, "SECURITY: Authorization failure - user=%s resource=%s resourceID=%s reason=%s",
		event.UserID, event.Resource, event.ResourceID, event.Reason)
}

// LogRateLimitExceeded logs a rate limit exceeded event
func (sl *SecurityLogger) LogRateLimitExceeded(ctx context.Context, userID, limitType string, current, max int) {
	event := SecurityEvent{
		Type:      SecurityEventRateLimitExceeded,
		UserID:    userID,
		Reason:    limitType,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"current": current,
			"max":     max,
		},
	}
	sl.logger.Warnf(ctx, "SECURITY: Rate limit exceeded - user=%s limit=%s current=%d max=%d",
		event.UserID, event.Reason, current, max)
}

// LogInvalidInput logs an invalid input event
func (sl *SecurityLogger) LogInvalidInput(ctx context.Context, userID, field, value, reason string) {
	event := SecurityEvent{
		Type:      SecurityEventInvalidInput,
		UserID:    userID,
		Reason:    reason,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"field": field,
			"value": value,
		},
	}
	sl.logger.Warnf(ctx, "SECURITY: Invalid input - user=%s field=%s reason=%s",
		event.UserID, field, event.Reason)
}

// LogSuspiciousActivity logs suspicious activity
func (sl *SecurityLogger) LogSuspiciousActivity(ctx context.Context, userID, activity string, metadata map[string]interface{}) {
	event := SecurityEvent{
		Type:      SecurityEventSuspiciousActivity,
		UserID:    userID,
		Reason:    activity,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	sl.logger.Warnf(ctx, "SECURITY: Suspicious activity - user=%s activity=%s",
		event.UserID, event.Reason)
}
