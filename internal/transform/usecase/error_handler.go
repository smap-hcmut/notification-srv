package usecase

import (
	"context"
	"fmt"
	"strings"
)

// HandleTransformError handles transformation errors with context
func (e *errorHandlerImpl) HandleTransformError(ctx context.Context, msgType, channel string, err error, payload string) {
	// Classify error type
	errorType := e.classifyTransformError(err)

	// Log error with context
	e.logger.Errorf(ctx, "Transform error for %s on channel %s: %v (type: %s)",
		msgType, channel, err, errorType)

	// Log payload for debugging (truncated for security)
	truncatedPayload := e.truncatePayload(payload)
	e.logger.Debugf(ctx, "Failed payload for %s: %s", msgType, truncatedPayload)

	// Update metrics
	e.metrics.IncrementTransformError(msgType, errorType)

	// Additional error-specific handling
	e.handleSpecificError(ctx, msgType, channel, errorType, err)
}

// HandleValidationError handles input validation errors
func (e *errorHandlerImpl) HandleValidationError(ctx context.Context, msgType, channel string, err error, payload string) {
	// Log validation error
	e.logger.Warnf(ctx, "Validation error for %s on channel %s: %v",
		msgType, channel, err)

	// Log payload for debugging (truncated for security)
	truncatedPayload := e.truncatePayload(payload)
	e.logger.Debugf(ctx, "Invalid payload for %s: %s", msgType, truncatedPayload)

	// Update metrics
	e.metrics.IncrementTransformError(msgType, "validation")

	// Handle specific validation errors
	e.handleValidationErrorType(ctx, msgType, channel, err)
}

// classifyTransformError classifies transform error into categories
func (e *errorHandlerImpl) classifyTransformError(err error) string {
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "json"):
		return "json_parse"
	case strings.Contains(errStr, "validation"):
		return "validation"
	case strings.Contains(errStr, "missing") && strings.Contains(errStr, "field"):
		return "missing_field"
	case strings.Contains(errStr, "invalid") && strings.Contains(errStr, "status"):
		return "invalid_status"
	case strings.Contains(errStr, "invalid") && strings.Contains(errStr, "platform"):
		return "invalid_platform"
	case strings.Contains(errStr, "invalid") && strings.Contains(errStr, "value"):
		return "invalid_value"
	case strings.Contains(errStr, "output validation"):
		return "output_validation"
	case strings.Contains(errStr, "transform"):
		return "transform"
	default:
		return "unknown"
	}
}

// handleSpecificError provides specific handling for different error types
func (e *errorHandlerImpl) handleSpecificError(ctx context.Context, msgType, channel string, errorType string, err error) {
	switch errorType {
	case "json_parse":
		e.logger.Errorf(ctx, "CRITICAL: Malformed JSON from publisher on %s - investigate message format", channel)

	case "validation":
		e.logger.Warnf(ctx, "Publisher validation error on %s - check message structure", channel)

	case "missing_field":
		e.logger.Warnf(ctx, "Publisher missing required field on %s - check message completeness", channel)

	case "invalid_status":
		e.logger.Warnf(ctx, "Invalid status value on %s - publisher using unsupported status", channel)

	case "invalid_platform":
		e.logger.Warnf(ctx, "Invalid platform value on %s - publisher using unsupported platform", channel)

	case "output_validation":
		e.logger.Errorf(ctx, "CRITICAL: Transform layer produced invalid output on %s - bug in transformer", channel)

	default:
		e.logger.Errorf(ctx, "Unhandled transform error type %s on %s", errorType, channel)
	}
}

// handleValidationErrorType provides specific handling for validation error types
func (e *errorHandlerImpl) handleValidationErrorType(ctx context.Context, msgType, channel string, err error) {
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "invalid topic format"):
		e.logger.Errorf(ctx, "CRITICAL: Invalid topic format %s - Redis channel pattern mismatch", channel)

	case strings.Contains(errStr, "unsupported topic type"):
		e.logger.Warnf(ctx, "Unsupported topic type in %s - new topic type not implemented", channel)

	case strings.Contains(errStr, "invalid") && strings.Contains(errStr, "id"):
		e.logger.Warnf(ctx, "Invalid ID format in %s - check ID validation rules", channel)

	default:
		e.logger.Warnf(ctx, "General validation error on %s: %v", channel, err)
	}
}

// truncatePayload truncates payload for logging while preserving structure visibility
func (e *errorHandlerImpl) truncatePayload(payload string) string {
	const maxLogLength = 200

	if len(payload) <= maxLogLength {
		return payload
	}

	// Try to keep JSON structure visible
	if strings.HasPrefix(payload, "{") && strings.HasSuffix(payload, "}") {
		middle := "...[truncated]..."
		start := payload[:50]
		end := payload[len(payload)-50:]
		return fmt.Sprintf("%s%s%s", start, middle, end)
	}

	// Simple truncation for non-JSON
	return fmt.Sprintf("%s...[truncated, total length: %d]", payload[:maxLogLength], len(payload))
}

// CreateErrorContext creates error context for detailed debugging
func (e *errorHandlerImpl) CreateErrorContext(msgType, channel string, err error, payload string) map[string]interface{} {
	return map[string]interface{}{
		"message_type":    msgType,
		"channel":         channel,
		"error":           err.Error(),
		"error_type":      e.classifyTransformError(err),
		"payload_length":  len(payload),
		"payload_preview": e.truncatePayload(payload),
		"timestamp":       "time.Now().UTC()", // Would use actual time
	}
}

// ShouldRetryTransform determines if transform should be retried based on error type
func (e *errorHandlerImpl) ShouldRetryTransform(err error) bool {
	errStr := strings.ToLower(err.Error())

	// Don't retry validation or structural errors
	retryableErrors := []string{
		"json_parse",
		"missing_field",
		"invalid_status",
		"invalid_platform",
		"invalid_value",
		"validation",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return false
		}
	}

	// Retry for unknown or temporary errors
	return true
}
