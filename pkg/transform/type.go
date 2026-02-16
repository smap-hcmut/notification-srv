package transform

import (
	"smap-websocket/pkg/log"
	"sync"
	"time"
)

// InputValidator implements MessageValidator interface
type InputValidator struct{}

// ErrorHandlerImpl implements the ErrorHandler interface
type ErrorHandlerImpl struct {
	logger  log.Logger
	metrics MetricsCollector
}

// MetricsCollectorImpl implements the MetricsCollector interface
type MetricsCollectorImpl struct {
	metrics          TransformMetrics
	mu               sync.RWMutex
	projectLatencies []time.Duration
	jobLatencies     []time.Duration
	latencyMu        sync.Mutex
	maxLatencySize   int
}

// MessageTransformerImpl implements the MessageTransformer interface
type MessageTransformerImpl struct {
	projectTransformer *ProjectTransformer
	jobTransformer     *JobTransformer
	validator          MessageValidator
	errorHandler       ErrorHandler
	logger             log.Logger
}

// ProjectTransformer handles transformation of project input messages to output format
type ProjectTransformer struct {
	validator MessageValidator
	metrics   MetricsCollector
	logger    log.Logger
}

// JobTransformer handles transformation of job input messages to output format
type JobTransformer struct {
	validator MessageValidator
	metrics   MetricsCollector
	logger    log.Logger
}

// TransformMetrics defines metrics collected by the transform layer
type TransformMetrics struct {
	ProjectTransformSuccess int64
	ProjectTransformErrors  int64
	JobTransformSuccess     int64
	JobTransformErrors      int64
	ProjectTransformLatency time.Duration
	JobTransformLatency     time.Duration
	ValidationErrors        int64
	ValidationSuccess       int64
	JSONParseErrors         int64
	MissingFieldErrors      int64
	InvalidStatusErrors     int64
	InvalidPlatformErrors   int64
	InvalidValueErrors      int64
}
