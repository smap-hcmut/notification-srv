package transform

import (
	"context"
	"testing"
	"time"

	"smap-websocket/internal/types"
)

// Mock implementations for testing
type mockValidator struct {
	validateProjectInputFunc func(payload string) error
	validateJobInputFunc     func(payload string) error
}

func (m *mockValidator) ValidateProjectInput(payload string) error {
	if m.validateProjectInputFunc != nil {
		return m.validateProjectInputFunc(payload)
	}
	return nil
}

func (m *mockValidator) ValidateJobInput(payload string) error {
	if m.validateJobInputFunc != nil {
		return m.validateJobInputFunc(payload)
	}
	return nil
}

type mockMetrics struct {
	transformSuccess map[string]int64
	transformErrors  map[string]map[string]int64
	latencies        map[string][]time.Duration
}

func newMockMetrics() *mockMetrics {
	return &mockMetrics{
		transformSuccess: make(map[string]int64),
		transformErrors:  make(map[string]map[string]int64),
		latencies:        make(map[string][]time.Duration),
	}
}

func (m *mockMetrics) IncrementTransformSuccess(msgType string) {
	m.transformSuccess[msgType]++
}

func (m *mockMetrics) IncrementTransformError(msgType, errorType string) {
	if m.transformErrors[msgType] == nil {
		m.transformErrors[msgType] = make(map[string]int64)
	}
	m.transformErrors[msgType][errorType]++
}

func (m *mockMetrics) RecordTransformLatency(msgType string, duration time.Duration) {
	m.latencies[msgType] = append(m.latencies[msgType], duration)
}

func (m *mockMetrics) GetMetrics() TransformMetrics {
	return TransformMetrics{}
}

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	// Store log for testing
}

func (m *mockLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	// Store log for testing
}

func (m *mockLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	// Store log for testing
}

func (m *mockLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	// Store log for testing
}

func TestProjectTransformer_Transform(t *testing.T) {
	validator := &mockValidator{}
	metrics := newMockMetrics()
	logger := &mockLogger{}

	transformer := NewProjectTransformer(validator, metrics, logger)

	tests := []struct {
		name        string
		payload     string
		projectID   string
		userID      string
		setupMocks  func()
		wantErr     bool
		errContains string
		checkResult func(t *testing.T, result *types.ProjectNotificationMessage)
	}{
		{
			name:      "valid project message with progress",
			payload:   `{"status": "PROCESSING", "progress": {"current": 50, "total": 100, "percentage": 50.0, "eta": 10.5, "errors": ["error1"]}}`,
			projectID: "proj_123",
			userID:    "user_456",
			setupMocks: func() {
				validator.validateProjectInputFunc = func(payload string) error {
					return nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *types.ProjectNotificationMessage) {
				if result.Status != types.ProjectStatusProcessing {
					t.Errorf("Status = %v, want %v", result.Status, types.ProjectStatusProcessing)
				}
				if result.Progress == nil {
					t.Error("Progress should not be nil")
				} else {
					if result.Progress.Current != 50 {
						t.Errorf("Progress.Current = %v, want %v", result.Progress.Current, 50)
					}
					if result.Progress.Total != 100 {
						t.Errorf("Progress.Total = %v, want %v", result.Progress.Total, 100)
					}
					if result.Progress.Percentage != 50.0 {
						t.Errorf("Progress.Percentage = %v, want %v", result.Progress.Percentage, 50.0)
					}
				}
			},
		},
		{
			name:      "valid project message without progress",
			payload:   `{"status": "COMPLETED"}`,
			projectID: "proj_123",
			userID:    "user_456",
			setupMocks: func() {
				validator.validateProjectInputFunc = func(payload string) error {
					return nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *types.ProjectNotificationMessage) {
				if result.Status != types.ProjectStatusCompleted {
					t.Errorf("Status = %v, want %v", result.Status, types.ProjectStatusCompleted)
				}
				if result.Progress != nil {
					t.Error("Progress should be nil for COMPLETED without progress")
				}
			},
		},
		{
			name:      "validation error",
			payload:   `{"status": "INVALID"}`,
			projectID: "proj_123",
			userID:    "user_456",
			setupMocks: func() {
				validator.validateProjectInputFunc = func(payload string) error {
					return types.ErrInvalidStatus("INVALID")
				}
			},
			wantErr:     true,
			errContains: "validation failed",
		},
		{
			name:      "invalid JSON",
			payload:   `{"status": "PROCESSING"`,
			projectID: "proj_123",
			userID:    "user_456",
			setupMocks: func() {
				validator.validateProjectInputFunc = func(payload string) error {
					return nil
				}
			},
			wantErr:     true,
			errContains: "failed to parse input message",
		},
		{
			name:      "percentage clamping",
			payload:   `{"status": "PROCESSING", "progress": {"current": 50, "total": 100, "percentage": 150.0, "eta": 10.5, "errors": []}}`,
			projectID: "proj_123",
			userID:    "user_456",
			setupMocks: func() {
				validator.validateProjectInputFunc = func(payload string) error {
					return nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *types.ProjectNotificationMessage) {
				if result.Progress.Percentage != 100.0 {
					t.Errorf("Progress.Percentage = %v, want %v (should be clamped to 100)", result.Progress.Percentage, 100.0)
				}
			},
		},
		{
			name:      "negative ETA clamping",
			payload:   `{"status": "PROCESSING", "progress": {"current": 50, "total": 100, "percentage": 50.0, "eta": -5.0, "errors": []}}`,
			projectID: "proj_123",
			userID:    "user_456",
			setupMocks: func() {
				validator.validateProjectInputFunc = func(payload string) error {
					return nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *types.ProjectNotificationMessage) {
				if result.Progress.ETA != 0.0 {
					t.Errorf("Progress.ETA = %v, want %v (should be clamped to 0)", result.Progress.ETA, 0.0)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			metrics.transformSuccess = make(map[string]int64)
			metrics.transformErrors = make(map[string]map[string]int64)

			tt.setupMocks()

			result, err := transformer.Transform(context.Background(), tt.payload, tt.projectID, tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("Transform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Transform() error = %v, want to contain %v", err.Error(), tt.errContains)
				}
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}

			// Check metrics
			if metrics.transformSuccess["project"] != 1 {
				t.Errorf("Expected 1 successful transform, got %d", metrics.transformSuccess["project"])
			}
		})
	}
}

func TestProjectTransformer_TransformProgress(t *testing.T) {
	validator := &mockValidator{}
	metrics := newMockMetrics()
	logger := &mockLogger{}

	transformer := NewProjectTransformer(validator, metrics, logger)

	tests := []struct {
		name        string
		input       *types.ProgressInput
		wantErr     bool
		checkOutput func(t *testing.T, output *types.Progress)
	}{
		{
			name: "normal progress",
			input: &types.ProgressInput{
				Current:    25,
				Total:      100,
				Percentage: 25.0,
				ETA:        15.5,
				Errors:     []string{"error1", "error2"},
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output *types.Progress) {
				if output.Current != 25 {
					t.Errorf("Current = %v, want %v", output.Current, 25)
				}
				if output.Total != 100 {
					t.Errorf("Total = %v, want %v", output.Total, 100)
				}
				if output.Percentage != 25.0 {
					t.Errorf("Percentage = %v, want %v", output.Percentage, 25.0)
				}
				if output.ETA != 15.5 {
					t.Errorf("ETA = %v, want %v", output.ETA, 15.5)
				}
				if len(output.Errors) != 2 {
					t.Errorf("Errors length = %v, want %v", len(output.Errors), 2)
				}
			},
		},
		{
			name: "percentage clamping - negative",
			input: &types.ProgressInput{
				Current:    0,
				Total:      100,
				Percentage: -10.0,
				ETA:        10.0,
				Errors:     []string{},
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output *types.Progress) {
				if output.Percentage != 0.0 {
					t.Errorf("Percentage = %v, want %v (should be clamped)", output.Percentage, 0.0)
				}
			},
		},
		{
			name: "percentage clamping - over 100",
			input: &types.ProgressInput{
				Current:    100,
				Total:      100,
				Percentage: 150.0,
				ETA:        0.0,
				Errors:     []string{},
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output *types.Progress) {
				if output.Percentage != 100.0 {
					t.Errorf("Percentage = %v, want %v (should be clamped)", output.Percentage, 100.0)
				}
			},
		},
		{
			name: "ETA clamping - negative",
			input: &types.ProgressInput{
				Current:    50,
				Total:      100,
				Percentage: 50.0,
				ETA:        -5.0,
				Errors:     []string{},
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output *types.Progress) {
				if output.ETA != 0.0 {
					t.Errorf("ETA = %v, want %v (should be clamped)", output.ETA, 0.0)
				}
			},
		},
		{
			name: "nil errors handling",
			input: &types.ProgressInput{
				Current:    50,
				Total:      100,
				Percentage: 50.0,
				ETA:        10.0,
				Errors:     nil,
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output *types.Progress) {
				if output.Errors == nil {
					t.Error("Errors should not be nil")
				}
				if len(output.Errors) != 0 {
					t.Errorf("Errors length = %v, want %v", len(output.Errors), 0)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := transformer.transformProgress(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("transformProgress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestProjectTransformer_TransformProgressSafe(t *testing.T) {
	validator := &mockValidator{}
	metrics := newMockMetrics()
	logger := &mockLogger{}

	transformer := NewProjectTransformer(validator, metrics, logger)

	tests := []struct {
		name        string
		input       *types.ProgressInput
		wantErr     bool
		errMsg      string
		checkOutput func(t *testing.T, output *types.Progress)
	}{
		{
			name: "valid input",
			input: &types.ProgressInput{
				Current:    50,
				Total:      100,
				Percentage: 50.0,
				ETA:        10.0,
				Errors:     []string{"error1"},
			},
			wantErr: false,
		},
		{
			name: "negative current",
			input: &types.ProgressInput{
				Current:    -1,
				Total:      100,
				Percentage: 50.0,
				ETA:        10.0,
			},
			wantErr: true,
			errMsg:  "invalid current value",
		},
		{
			name: "negative total",
			input: &types.ProgressInput{
				Current:    50,
				Total:      -1,
				Percentage: 50.0,
				ETA:        10.0,
			},
			wantErr: true,
			errMsg:  "invalid total value",
		},
		{
			name: "current exceeds total",
			input: &types.ProgressInput{
				Current:    150,
				Total:      100,
				Percentage: 50.0,
				ETA:        10.0,
			},
			wantErr: true,
			errMsg:  "invalid current value",
		},
		{
			name: "percentage calculation when invalid input",
			input: &types.ProgressInput{
				Current:    25,
				Total:      100,
				Percentage: -50.0, // Invalid percentage, should be calculated
				ETA:        10.0,
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output *types.Progress) {
				expectedPercentage := 25.0 // 25/100 * 100
				if output.Percentage != expectedPercentage {
					t.Errorf("Percentage = %v, want %v (should be calculated)", output.Percentage, expectedPercentage)
				}
			},
		},
		{
			name: "zero total percentage calculation",
			input: &types.ProgressInput{
				Current:    0,
				Total:      0,
				Percentage: 50.0, // Should be overridden to 0
				ETA:        10.0,
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output *types.Progress) {
				if output.Percentage != 0.0 {
					t.Errorf("Percentage = %v, want %v (should be 0 for zero total)", output.Percentage, 0.0)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := transformer.transformProgressSafe(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("transformProgressSafe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("transformProgressSafe() error = %v, want to contain %v", err.Error(), tt.errMsg)
				}
				return
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

// ============================================================================
// Phase-Based Progress Transform Tests
// ============================================================================

func TestProjectTransformer_TransformPhaseBased(t *testing.T) {
	validator := &mockValidator{}
	metrics := newMockMetrics()
	logger := &mockLogger{}

	transformer := NewProjectTransformer(validator, metrics, logger)

	tests := []struct {
		name        string
		payload     string
		projectID   string
		userID      string
		wantErr     bool
		errContains string
		checkResult func(t *testing.T, result *types.ProjectPhaseNotificationMessage)
	}{
		{
			name: "valid project_progress message",
			payload: `{
				"type": "project_progress",
				"payload": {
					"project_id": "proj_xyz",
					"status": "PROCESSING",
					"crawl": {"total": 100, "done": 80, "errors": 2, "progress_percent": 82.0},
					"analyze": {"total": 78, "done": 45, "errors": 1, "progress_percent": 59.0},
					"overall_progress_percent": 70.5
				}
			}`,
			projectID: "proj_xyz",
			userID:    "user_123",
			wantErr:   false,
			checkResult: func(t *testing.T, result *types.ProjectPhaseNotificationMessage) {
				if result.Type != "project_progress" {
					t.Errorf("Type = %v, want %v", result.Type, "project_progress")
				}
				if result.Payload.ProjectID != "proj_xyz" {
					t.Errorf("ProjectID = %v, want %v", result.Payload.ProjectID, "proj_xyz")
				}
				if result.Payload.Status != "PROCESSING" {
					t.Errorf("Status = %v, want %v", result.Payload.Status, "PROCESSING")
				}
				if result.Payload.Crawl == nil {
					t.Error("Crawl should not be nil")
				} else {
					if result.Payload.Crawl.Done != 80 {
						t.Errorf("Crawl.Done = %v, want %v", result.Payload.Crawl.Done, 80)
					}
				}
				if result.Payload.Analyze == nil {
					t.Error("Analyze should not be nil")
				} else {
					if result.Payload.Analyze.Done != 45 {
						t.Errorf("Analyze.Done = %v, want %v", result.Payload.Analyze.Done, 45)
					}
				}
				if result.Payload.OverallProgressPercent != 70.5 {
					t.Errorf("OverallProgressPercent = %v, want %v", result.Payload.OverallProgressPercent, 70.5)
				}
			},
		},
		{
			name: "valid project_completed message",
			payload: `{
				"type": "project_completed",
				"payload": {
					"project_id": "proj_xyz",
					"status": "DONE",
					"crawl": {"total": 100, "done": 98, "errors": 2, "progress_percent": 100.0},
					"analyze": {"total": 98, "done": 95, "errors": 3, "progress_percent": 100.0},
					"overall_progress_percent": 100.0
				}
			}`,
			projectID: "proj_xyz",
			userID:    "user_123",
			wantErr:   false,
			checkResult: func(t *testing.T, result *types.ProjectPhaseNotificationMessage) {
				if result.Type != "project_completed" {
					t.Errorf("Type = %v, want %v", result.Type, "project_completed")
				}
				if result.Payload.Status != "DONE" {
					t.Errorf("Status = %v, want %v", result.Payload.Status, "DONE")
				}
				if result.Payload.OverallProgressPercent != 100.0 {
					t.Errorf("OverallProgressPercent = %v, want %v", result.Payload.OverallProgressPercent, 100.0)
				}
			},
		},
		{
			name: "valid with INITIALIZING status",
			payload: `{
				"type": "project_progress",
				"payload": {
					"project_id": "proj_xyz",
					"status": "INITIALIZING",
					"overall_progress_percent": 0
				}
			}`,
			projectID: "proj_xyz",
			userID:    "user_123",
			wantErr:   false,
			checkResult: func(t *testing.T, result *types.ProjectPhaseNotificationMessage) {
				if result.Payload.Status != "INITIALIZING" {
					t.Errorf("Status = %v, want %v", result.Payload.Status, "INITIALIZING")
				}
				if result.Payload.Crawl != nil {
					t.Error("Crawl should be nil for INITIALIZING")
				}
			},
		},
		{
			name: "valid with FAILED status",
			payload: `{
				"type": "project_completed",
				"payload": {
					"project_id": "proj_xyz",
					"status": "FAILED",
					"overall_progress_percent": 50.0
				}
			}`,
			projectID: "proj_xyz",
			userID:    "user_123",
			wantErr:   false,
			checkResult: func(t *testing.T, result *types.ProjectPhaseNotificationMessage) {
				if result.Payload.Status != "FAILED" {
					t.Errorf("Status = %v, want %v", result.Payload.Status, "FAILED")
				}
			},
		},
		{
			name: "invalid status",
			payload: `{
				"type": "project_progress",
				"payload": {
					"project_id": "proj_xyz",
					"status": "INVALID_STATUS",
					"overall_progress_percent": 50.0
				}
			}`,
			projectID:   "proj_xyz",
			userID:      "user_123",
			wantErr:     true,
			errContains: "validation failed",
		},
		{
			name: "missing project_id",
			payload: `{
				"type": "project_progress",
				"payload": {
					"status": "PROCESSING",
					"overall_progress_percent": 50.0
				}
			}`,
			projectID:   "proj_xyz",
			userID:      "user_123",
			wantErr:     true,
			errContains: "validation failed",
		},
		{
			name: "invalid type",
			payload: `{
				"type": "invalid_type",
				"payload": {
					"project_id": "proj_xyz",
					"status": "PROCESSING"
				}
			}`,
			projectID:   "proj_xyz",
			userID:      "user_123",
			wantErr:     true,
			errContains: "validation failed",
		},
		{
			name:        "invalid JSON",
			payload:     `{invalid json`,
			projectID:   "proj_xyz",
			userID:      "user_123",
			wantErr:     true,
			errContains: "failed to parse",
		},
		{
			name: "invalid progress percent in phase (validation rejects > 100)",
			payload: `{
				"type": "project_progress",
				"payload": {
					"project_id": "proj_xyz",
					"status": "PROCESSING",
					"crawl": {"total": 100, "done": 80, "errors": 0, "progress_percent": 150.0},
					"overall_progress_percent": 70.0
				}
			}`,
			projectID:   "proj_xyz",
			userID:      "user_123",
			wantErr:     true,
			errContains: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset metrics
			metrics.transformSuccess = make(map[string]int64)
			metrics.transformErrors = make(map[string]map[string]int64)

			result, err := transformer.TransformPhaseBased(context.Background(), tt.payload, tt.projectID, tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("TransformPhaseBased() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("TransformPhaseBased() error = %v, want to contain %v", err.Error(), tt.errContains)
				}
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}

			// Check metrics
			if metrics.transformSuccess["project_phase"] != 1 {
				t.Errorf("Expected 1 successful transform, got %d", metrics.transformSuccess["project_phase"])
			}
		})
	}
}

func TestProjectTransformer_TransformAny(t *testing.T) {
	validator := &mockValidator{}
	metrics := newMockMetrics()
	logger := &mockLogger{}

	transformer := NewProjectTransformer(validator, metrics, logger)

	tests := []struct {
		name           string
		payload        string
		projectID      string
		userID         string
		setupMocks     func()
		wantErr        bool
		wantPhaseBased bool
	}{
		{
			name: "routes to phase-based for project_progress",
			payload: `{
				"type": "project_progress",
				"payload": {
					"project_id": "proj_xyz",
					"status": "PROCESSING",
					"overall_progress_percent": 50.0
				}
			}`,
			projectID:      "proj_xyz",
			userID:         "user_123",
			setupMocks:     func() {},
			wantErr:        false,
			wantPhaseBased: true,
		},
		{
			name: "routes to phase-based for project_completed",
			payload: `{
				"type": "project_completed",
				"payload": {
					"project_id": "proj_xyz",
					"status": "DONE",
					"overall_progress_percent": 100.0
				}
			}`,
			projectID:      "proj_xyz",
			userID:         "user_123",
			setupMocks:     func() {},
			wantErr:        false,
			wantPhaseBased: true,
		},
		{
			name:      "routes to legacy for old format",
			payload:   `{"status": "PROCESSING", "progress": {"current": 50, "total": 100, "percentage": 50.0, "eta": 10.0, "errors": []}}`,
			projectID: "proj_xyz",
			userID:    "user_123",
			setupMocks: func() {
				validator.validateProjectInputFunc = func(payload string) error {
					return nil
				}
			},
			wantErr:        false,
			wantPhaseBased: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset
			metrics.transformSuccess = make(map[string]int64)
			tt.setupMocks()

			result, err := transformer.TransformAny(context.Background(), tt.payload, tt.projectID, tt.userID)

			if (err != nil) != tt.wantErr {
				t.Errorf("TransformAny() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Check result type
			_, isPhaseBased := result.(*types.ProjectPhaseNotificationMessage)
			if isPhaseBased != tt.wantPhaseBased {
				t.Errorf("TransformAny() returned phase-based = %v, want %v", isPhaseBased, tt.wantPhaseBased)
			}
		})
	}
}
