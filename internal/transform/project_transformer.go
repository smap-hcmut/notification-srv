package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smap-websocket/internal/types"
)

// ProjectTransformer handles transformation of project input messages to output format
type ProjectTransformer struct {
	validator MessageValidator
	metrics   MetricsCollector
	logger    Logger
}

// NewProjectTransformer creates a new project message transformer
func NewProjectTransformer(validator MessageValidator, metrics MetricsCollector, logger Logger) *ProjectTransformer {
	return &ProjectTransformer{
		validator: validator,
		metrics:   metrics,
		logger:    logger,
	}
}

// Transform transforms project input message to standardized output format
func (t *ProjectTransformer) Transform(ctx context.Context, payload string, projectID, userID string) (*types.ProjectNotificationMessage, error) {
	startTime := time.Now()

	// Validate input first
	if err := t.validator.ValidateProjectInput(payload); err != nil {
		t.metrics.IncrementTransformError("project", "validation")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Parse input message
	var inputMsg types.ProjectInputMessage
	if err := json.Unmarshal([]byte(payload), &inputMsg); err != nil {
		t.metrics.IncrementTransformError("project", "json_parse")
		return nil, fmt.Errorf("failed to parse input message: %w", err)
	}

	// Transform to output format
	outputMsg, err := t.transformToOutput(&inputMsg)
	if err != nil {
		t.metrics.IncrementTransformError("project", "transform")
		return nil, fmt.Errorf("failed to transform message: %w", err)
	}

	// Validate output
	if err := outputMsg.Validate(); err != nil {
		t.metrics.IncrementTransformError("project", "output_validation")
		return nil, fmt.Errorf("output validation failed: %w", err)
	}

	// Record success metrics
	duration := time.Since(startTime)
	t.metrics.RecordTransformLatency("project", duration)
	t.metrics.IncrementTransformSuccess("project")

	t.logger.Debugf(ctx, "Successfully transformed project message for project %s, user %s in %v",
		projectID, userID, duration)

	return outputMsg, nil
}

// transformToOutput converts input message to output format
func (t *ProjectTransformer) transformToOutput(input *types.ProjectInputMessage) (*types.ProjectNotificationMessage, error) {
	// Create output message
	output := &types.ProjectNotificationMessage{
		Status: types.ProjectStatus(input.Status),
	}

	// Transform progress if present
	if input.Progress != nil {
		progress, err := t.transformProgress(input.Progress)
		if err != nil {
			return nil, fmt.Errorf("failed to transform progress: %w", err)
		}
		output.Progress = progress
	}

	return output, nil
}

// transformProgress transforms progress input to output format
func (t *ProjectTransformer) transformProgress(input *types.ProgressInput) (*types.Progress, error) {
	// Clamp percentage to valid range
	percentage := input.Percentage
	if percentage < 0 {
		percentage = 0
		t.logger.Warnf(context.Background(), "Clamped negative percentage %f to 0", input.Percentage)
	} else if percentage > 100 {
		percentage = 100
		t.logger.Warnf(context.Background(), "Clamped excessive percentage %f to 100", input.Percentage)
	}

	// Ensure ETA is non-negative
	eta := input.ETA
	if eta < 0 {
		eta = 0
		t.logger.Warnf(context.Background(), "Clamped negative ETA %f to 0", input.ETA)
	}

	// Copy errors slice to prevent shared references
	errors := make([]string, 0)
	if input.Errors != nil {
		errors = make([]string, len(input.Errors))
		copy(errors, input.Errors)
	}

	// Create output progress
	return &types.Progress{
		Current:    input.Current,
		Total:      input.Total,
		Percentage: percentage,
		ETA:        eta,
		Errors:     errors,
	}, nil
}

// transformProgressSafe safely transforms progress with validation
func (t *ProjectTransformer) transformProgressSafe(input *types.ProgressInput) (*types.Progress, error) {
	// Validate input first
	if input.Current < 0 {
		return nil, fmt.Errorf("invalid current value: %d (must be non-negative)", input.Current)
	}

	if input.Total < 0 {
		return nil, fmt.Errorf("invalid total value: %d (must be non-negative)", input.Total)
	}

	if input.Current > input.Total {
		return nil, fmt.Errorf("invalid current value: %d exceeds total %d", input.Current, input.Total)
	}

	// Calculate percentage if not provided or invalid
	var percentage float64
	if input.Total > 0 {
		calculatedPercentage := float64(input.Current) / float64(input.Total) * 100

		// Use calculated percentage if input percentage is invalid
		if input.Percentage < 0 || input.Percentage > 100 {
			percentage = calculatedPercentage
			t.logger.Warnf(context.Background(),
				"Invalid percentage %f, using calculated value %f",
				input.Percentage, calculatedPercentage)
		} else {
			percentage = input.Percentage
		}
	} else {
		percentage = 0
	}

	// Ensure ETA is non-negative
	eta := input.ETA
	if eta < 0 {
		eta = 0
		t.logger.Warnf(context.Background(), "Clamped negative ETA %f to 0", input.ETA)
	}

	// Copy errors slice to prevent shared references
	errors := make([]string, 0)
	if input.Errors != nil {
		errors = make([]string, len(input.Errors))
		copy(errors, input.Errors)
	}

	return &types.Progress{
		Current:    input.Current,
		Total:      input.Total,
		Percentage: percentage,
		ETA:        eta,
		Errors:     errors,
	}, nil
}

// ============================================================================
// Phase-Based Progress Transform (New Format)
// ============================================================================

// TransformPhaseBased transforms phase-based project input message to output format
// This handles the new message format with crawl/analyze phases
func (t *ProjectTransformer) TransformPhaseBased(ctx context.Context, payload string, projectID, userID string) (*types.ProjectPhaseNotificationMessage, error) {
	startTime := time.Now()

	// Parse input message
	var inputMsg types.ProjectPhaseInputMessage
	if err := json.Unmarshal([]byte(payload), &inputMsg); err != nil {
		t.metrics.IncrementTransformError("project_phase", "json_parse")
		return nil, fmt.Errorf("failed to parse phase-based input message: %w", err)
	}

	// Validate input
	if err := inputMsg.Validate(); err != nil {
		t.metrics.IncrementTransformError("project_phase", "validation")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Transform to output format
	outputMsg := t.transformPhaseToOutput(&inputMsg)

	// Validate output
	if err := outputMsg.Validate(); err != nil {
		t.metrics.IncrementTransformError("project_phase", "output_validation")
		return nil, fmt.Errorf("output validation failed: %w", err)
	}

	// Record success metrics
	duration := time.Since(startTime)
	t.metrics.RecordTransformLatency("project_phase", duration)
	t.metrics.IncrementTransformSuccess("project_phase")

	t.logger.Debugf(ctx, "Successfully transformed phase-based project message for project %s, user %s in %v",
		projectID, userID, duration)

	return outputMsg, nil
}

// transformPhaseToOutput converts phase-based input to output format
func (t *ProjectTransformer) transformPhaseToOutput(input *types.ProjectPhaseInputMessage) *types.ProjectPhaseNotificationMessage {
	output := &types.ProjectPhaseNotificationMessage{
		Type: input.Type,
		Payload: types.ProjectPhasePayloadOutput{
			ProjectID:              input.Payload.ProjectID,
			Status:                 input.Payload.Status,
			OverallProgressPercent: input.Payload.OverallProgressPercent,
		},
	}

	// Transform crawl phase if present
	if input.Payload.Crawl != nil {
		output.Payload.Crawl = t.transformPhaseProgress(input.Payload.Crawl)
	}

	// Transform analyze phase if present
	if input.Payload.Analyze != nil {
		output.Payload.Analyze = t.transformPhaseProgress(input.Payload.Analyze)
	}

	return output
}

// transformPhaseProgress transforms phase progress input to output
func (t *ProjectTransformer) transformPhaseProgress(input *types.PhaseProgressInput) *types.PhaseProgress {
	// Clamp progress_percent to valid range
	progressPercent := input.ProgressPercent
	if progressPercent < 0 {
		progressPercent = 0
		t.logger.Warnf(context.Background(), "Clamped negative progress_percent %f to 0", input.ProgressPercent)
	} else if progressPercent > 100 {
		progressPercent = 100
		t.logger.Warnf(context.Background(), "Clamped excessive progress_percent %f to 100", input.ProgressPercent)
	}

	return &types.PhaseProgress{
		Total:           input.Total,
		Done:            input.Done,
		Errors:          input.Errors,
		ProgressPercent: progressPercent,
	}
}

// TransformAny transforms any project message (legacy or phase-based) based on payload format
// Returns either *types.ProjectNotificationMessage or *types.ProjectPhaseNotificationMessage
func (t *ProjectTransformer) TransformAny(ctx context.Context, payload string, projectID, userID string) (interface{}, error) {
	// Check if this is phase-based format
	if types.IsPhaseBasedMessage([]byte(payload)) {
		return t.TransformPhaseBased(ctx, payload, projectID, userID)
	}

	// Legacy format handling
	return t.Transform(ctx, payload, projectID, userID)
}
