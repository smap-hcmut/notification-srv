package usecase

import (
	"sync"
	"time"

	"notification-srv/internal/transform"
	"notification-srv/pkg/log"
)

// inputValidator implements transform.MessageValidator interface
type inputValidator struct{}

// errorHandlerImpl implements the transform.ErrorHandler interface
type errorHandlerImpl struct {
	logger  log.Logger
	metrics transform.MetricsCollector
}

// metricsCollectorImpl implements the transform.MetricsCollector interface
type metricsCollectorImpl struct {
	metrics          transform.TransformMetrics
	mu               sync.RWMutex
	projectLatencies []time.Duration
	jobLatencies     []time.Duration
	latencyMu        sync.Mutex
	maxLatencySize   int
}

// messageTransformerImpl implements the transform.MessageTransformer interface
type messageTransformerImpl struct {
	projectTransformer *projectTransformer
	jobTransformer     *jobTransformer
	validator          transform.MessageValidator
	errorHandler       transform.ErrorHandler
	logger             log.Logger
}

// projectTransformer handles transformation of project input messages to output format
type projectTransformer struct {
	validator transform.MessageValidator
	metrics   transform.MetricsCollector
	logger    log.Logger
}

// jobTransformer handles transformation of job input messages to output format
type jobTransformer struct {
	validator transform.MessageValidator
	metrics   transform.MetricsCollector
	logger    log.Logger
}
