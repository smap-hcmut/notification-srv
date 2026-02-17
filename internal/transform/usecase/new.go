package usecase

import (
	"time"

	"notification-srv/internal/transform"
	"notification-srv/pkg/log"
)

// NewMessageTransformer creates a new message transformer
func NewMessageTransformer(
	validator transform.MessageValidator,
	metrics transform.MetricsCollector,
	errorHandler transform.ErrorHandler,
	logger log.Logger,
) transform.MessageTransformer {
	return &messageTransformerImpl{
		projectTransformer: newProjectTransformer(validator, metrics, logger),
		jobTransformer:     newJobTransformer(validator, metrics, logger),
		validator:          validator,
		errorHandler:       errorHandler,
		logger:             logger,
	}
}

// newProjectTransformer creates a new project message transformer (private factory)
func newProjectTransformer(validator transform.MessageValidator, metrics transform.MetricsCollector, logger log.Logger) *projectTransformer {
	return &projectTransformer{
		validator: validator,
		metrics:   metrics,
		logger:    logger,
	}
}

// newJobTransformer creates a new job message transformer (private factory)
func newJobTransformer(validator transform.MessageValidator, metrics transform.MetricsCollector, logger log.Logger) *jobTransformer {
	return &jobTransformer{
		validator: validator,
		metrics:   metrics,
		logger:    logger,
	}
}

// NewInputValidator creates a new input validator
func NewInputValidator() transform.MessageValidator {
	return &inputValidator{}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger log.Logger, metrics transform.MetricsCollector) transform.ErrorHandler {
	return &errorHandlerImpl{
		logger:  logger,
		metrics: metrics,
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() transform.MetricsCollector {
	return &metricsCollectorImpl{
		maxLatencySize:   1000,
		projectLatencies: make([]time.Duration, 0, 1000),
		jobLatencies:     make([]time.Duration, 0, 1000),
	}
}
