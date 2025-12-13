package transform

import (
	"context"
	"time"

	"smap-websocket/internal/types"
)

// MessageTransformer defines the interface for transforming messages
type MessageTransformer interface {
	// TransformProjectMessage transforms a project input message to output format
	TransformProjectMessage(ctx context.Context, payload string, projectID, userID string) (*types.ProjectNotificationMessage, error)
	
	// TransformJobMessage transforms a job input message to output format
	TransformJobMessage(ctx context.Context, payload string, jobID, userID string) (*types.JobNotificationMessage, error)
	
	// TransformMessage transforms any message based on topic type
	TransformMessage(ctx context.Context, channel string, payload string) (interface{}, error)
}

// MessageValidator defines the interface for validating input messages
type MessageValidator interface {
	// ValidateProjectInput validates project input message structure
	ValidateProjectInput(payload string) error
	
	// ValidateJobInput validates job input message structure
	ValidateJobInput(payload string) error
}

// TransformMetrics defines metrics collected by the transform layer
type TransformMetrics struct {
	// Transform success/failure counts by type
	ProjectTransformSuccess int64
	ProjectTransformErrors  int64
	JobTransformSuccess     int64
	JobTransformErrors      int64
	
	// Transform latency metrics
	ProjectTransformLatency time.Duration
	JobTransformLatency     time.Duration
	
	// Validation metrics
	ValidationErrors   int64
	ValidationSuccess  int64
	
	// Error breakdown by type
	JSONParseErrors      int64
	MissingFieldErrors   int64
	InvalidStatusErrors  int64
	InvalidPlatformErrors int64
	InvalidValueErrors   int64
}

// MetricsCollector defines interface for collecting transform metrics
type MetricsCollector interface {
	// IncrementTransformSuccess increments successful transform count
	IncrementTransformSuccess(msgType string)
	
	// IncrementTransformError increments transform error count
	IncrementTransformError(msgType, errorType string)
	
	// RecordTransformLatency records transform processing time
	RecordTransformLatency(msgType string, duration time.Duration)
	
	// GetMetrics returns current metrics snapshot
	GetMetrics() TransformMetrics
}

// ErrorHandler defines interface for handling transform errors
type ErrorHandler interface {
	// HandleTransformError handles transformation errors with context
	HandleTransformError(ctx context.Context, msgType, channel string, err error, payload string)
	
	// HandleValidationError handles input validation errors
	HandleValidationError(ctx context.Context, msgType, channel string, err error, payload string)
}

// Logger defines interface for transform layer logging
type Logger interface {
	// Infof logs info level message with format
	Infof(ctx context.Context, format string, args ...interface{})
	
	// Warnf logs warn level message with format  
	Warnf(ctx context.Context, format string, args ...interface{})
	
	// Errorf logs error level message with format
	Errorf(ctx context.Context, format string, args ...interface{})
	
	// Debugf logs debug level message with format
	Debugf(ctx context.Context, format string, args ...interface{})
}