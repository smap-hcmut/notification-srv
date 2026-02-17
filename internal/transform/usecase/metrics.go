package usecase

import (
	"sync/atomic"
	"time"

	"notification-srv/internal/transform"
)

// IncrementTransformSuccess increments successful transform count
func (m *metricsCollectorImpl) IncrementTransformSuccess(msgType string) {
	switch msgType {
	case "project":
		atomic.AddInt64(&m.metrics.ProjectTransformSuccess, 1)
	case "job":
		atomic.AddInt64(&m.metrics.JobTransformSuccess, 1)
	}
	atomic.AddInt64(&m.metrics.ValidationSuccess, 1)
}

// IncrementTransformError increments transform error count
func (m *metricsCollectorImpl) IncrementTransformError(msgType, errorType string) {
	// Increment by message type
	switch msgType {
	case "project":
		atomic.AddInt64(&m.metrics.ProjectTransformErrors, 1)
	case "job":
		atomic.AddInt64(&m.metrics.JobTransformErrors, 1)
	}

	// Increment by error type
	switch errorType {
	case "validation":
		atomic.AddInt64(&m.metrics.ValidationErrors, 1)
	case "json_parse":
		atomic.AddInt64(&m.metrics.JSONParseErrors, 1)
	case "missing_field":
		atomic.AddInt64(&m.metrics.MissingFieldErrors, 1)
	case "invalid_status":
		atomic.AddInt64(&m.metrics.InvalidStatusErrors, 1)
	case "invalid_platform":
		atomic.AddInt64(&m.metrics.InvalidPlatformErrors, 1)
	case "invalid_value":
		atomic.AddInt64(&m.metrics.InvalidValueErrors, 1)
	}
}

// RecordTransformLatency records transform processing time
func (m *metricsCollectorImpl) RecordTransformLatency(msgType string, duration time.Duration) {
	m.latencyMu.Lock()
	defer m.latencyMu.Unlock()

	switch msgType {
	case "project":
		m.addProjectLatency(duration)
		// Update average
		m.mu.Lock()
		m.metrics.ProjectTransformLatency = m.calculateAverageProjectLatency()
		m.mu.Unlock()

	case "job":
		m.addJobLatency(duration)
		// Update average
		m.mu.Lock()
		m.metrics.JobTransformLatency = m.calculateAverageJobLatency()
		m.mu.Unlock()
	}
}

// addProjectLatency adds project latency measurement with size limit
func (m *metricsCollectorImpl) addProjectLatency(duration time.Duration) {
	if len(m.projectLatencies) >= m.maxLatencySize {
		// Remove oldest measurement (shift array)
		copy(m.projectLatencies, m.projectLatencies[1:])
		m.projectLatencies = m.projectLatencies[:len(m.projectLatencies)-1]
	}
	m.projectLatencies = append(m.projectLatencies, duration)
}

// addJobLatency adds job latency measurement with size limit
func (m *metricsCollectorImpl) addJobLatency(duration time.Duration) {
	if len(m.jobLatencies) >= m.maxLatencySize {
		// Remove oldest measurement (shift array)
		copy(m.jobLatencies, m.jobLatencies[1:])
		m.jobLatencies = m.jobLatencies[:len(m.jobLatencies)-1]
	}
	m.jobLatencies = append(m.jobLatencies, duration)
}

// calculateAverageProjectLatency calculates average project transform latency
func (m *metricsCollectorImpl) calculateAverageProjectLatency() time.Duration {
	if len(m.projectLatencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, lat := range m.projectLatencies {
		total += lat
	}

	return total / time.Duration(len(m.projectLatencies))
}

// calculateAverageJobLatency calculates average job transform latency
func (m *metricsCollectorImpl) calculateAverageJobLatency() time.Duration {
	if len(m.jobLatencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, lat := range m.jobLatencies {
		total += lat
	}

	return total / time.Duration(len(m.jobLatencies))
}

// GetMetrics returns current metrics snapshot
func (m *metricsCollectorImpl) GetMetrics() transform.TransformMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create copy to avoid race conditions
	return transform.TransformMetrics{
		ProjectTransformSuccess: atomic.LoadInt64(&m.metrics.ProjectTransformSuccess),
		ProjectTransformErrors:  atomic.LoadInt64(&m.metrics.ProjectTransformErrors),
		JobTransformSuccess:     atomic.LoadInt64(&m.metrics.JobTransformSuccess),
		JobTransformErrors:      atomic.LoadInt64(&m.metrics.JobTransformErrors),
		ProjectTransformLatency: m.metrics.ProjectTransformLatency,
		JobTransformLatency:     m.metrics.JobTransformLatency,
		ValidationErrors:        atomic.LoadInt64(&m.metrics.ValidationErrors),
		ValidationSuccess:       atomic.LoadInt64(&m.metrics.ValidationSuccess),
		JSONParseErrors:         atomic.LoadInt64(&m.metrics.JSONParseErrors),
		MissingFieldErrors:      atomic.LoadInt64(&m.metrics.MissingFieldErrors),
		InvalidStatusErrors:     atomic.LoadInt64(&m.metrics.InvalidStatusErrors),
		InvalidPlatformErrors:   atomic.LoadInt64(&m.metrics.InvalidPlatformErrors),
		InvalidValueErrors:      atomic.LoadInt64(&m.metrics.InvalidValueErrors),
	}
}

// Reset resets all metrics (useful for testing)
func (m *metricsCollectorImpl) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = transform.TransformMetrics{}

	m.latencyMu.Lock()
	m.projectLatencies = m.projectLatencies[:0]
	m.jobLatencies = m.jobLatencies[:0]
	m.latencyMu.Unlock()
}

// GetTransformSuccessRate returns the success rate for transforms
func (m *metricsCollectorImpl) GetTransformSuccessRate(msgType string) float64 {
	metrics := m.GetMetrics()

	var success, errors int64
	switch msgType {
	case "project":
		success = metrics.ProjectTransformSuccess
		errors = metrics.ProjectTransformErrors
	case "job":
		success = metrics.JobTransformSuccess
		errors = metrics.JobTransformErrors
	default:
		return 0.0
	}

	total := success + errors
	if total == 0 {
		return 0.0
	}

	return float64(success) / float64(total) * 100.0
}

// GetLatencyPercentile returns latency percentile for given message type
func (m *metricsCollectorImpl) GetLatencyPercentile(msgType string, percentile float64) time.Duration {
	m.latencyMu.Lock()
	defer m.latencyMu.Unlock()

	var latencies []time.Duration
	switch msgType {
	case "project":
		latencies = m.projectLatencies
	case "job":
		latencies = m.jobLatencies
	default:
		return 0
	}

	if len(latencies) == 0 {
		return 0
	}

	// Simple percentile calculation (would use sort in production)
	index := int(float64(len(latencies)) * percentile / 100.0)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	return latencies[index]
}
