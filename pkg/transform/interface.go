package transform

import (
	"context"
	"time"

	"smap-websocket/internal/types"
	"smap-websocket/pkg/log"
)

// MessageTransformer defines the interface for transforming messages
type MessageTransformer interface {
	TransformProjectMessage(ctx context.Context, payload string, projectID, userID string) (*types.ProjectNotificationMessage, error)
	TransformJobMessage(ctx context.Context, payload string, jobID, userID string) (*types.JobNotificationMessage, error)
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

// NewMessageTransformer creates a new message transformer
func NewMessageTransformer(
	validator MessageValidator,
	metrics MetricsCollector,
	errorHandler ErrorHandler,
	logger log.Logger,
) *MessageTransformerImpl {
	return &MessageTransformerImpl{
		projectTransformer: NewProjectTransformer(validator, metrics, logger),
		jobTransformer:     NewJobTransformer(validator, metrics, logger),
		validator:          validator,
		errorHandler:       errorHandler,
		logger:             logger,
	}
}

// NewProjectTransformer creates a new project message transformer
func NewProjectTransformer(validator MessageValidator, metrics MetricsCollector, logger log.Logger) *ProjectTransformer {
	return &ProjectTransformer{
		validator: validator,
		metrics:   metrics,
		logger:    logger,
	}
}

// NewJobTransformer creates a new job message transformer
func NewJobTransformer(validator MessageValidator, metrics MetricsCollector, logger log.Logger) *JobTransformer {
	return &JobTransformer{
		validator: validator,
		metrics:   metrics,
		logger:    logger,
	}
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger log.Logger, metrics MetricsCollector) *ErrorHandlerImpl {
	return &ErrorHandlerImpl{
		logger:  logger,
		metrics: metrics,
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollectorImpl {
	return &MetricsCollectorImpl{
		maxLatencySize:   1000,
		projectLatencies: make([]time.Duration, 0, 1000),
		jobLatencies:     make([]time.Duration, 0, 1000),
	}
}
