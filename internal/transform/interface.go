package transform

import (
	"context"
	"time"
)

// MessageTransformer defines the interface for transforming messages
type MessageTransformer interface {
	TransformProjectMessage(ctx context.Context, payload string, projectID, userID string) (*ProjectNotificationMessage, error)
	TransformJobMessage(ctx context.Context, payload string, jobID, userID string) (*JobNotificationMessage, error)
	TransformMessage(ctx context.Context, channel string, payload string) (interface{}, error)
}

// MessageValidator defines the interface for validating input messages
type MessageValidator interface {
	ValidateProjectInput(payload string) error
	ValidateJobInput(payload string) error
}

// MetricsCollector defines interface for collecting transform metrics
type MetricsCollector interface {
	IncrementTransformSuccess(msgType string)
	IncrementTransformError(msgType, errorType string)
	RecordTransformLatency(msgType string, duration time.Duration)
	GetMetrics() TransformMetrics
}

// ErrorHandler defines interface for handling transform errors
type ErrorHandler interface {
	HandleTransformError(ctx context.Context, msgType, channel string, err error, payload string)
	HandleValidationError(ctx context.Context, msgType, channel string, err error, payload string)
}
