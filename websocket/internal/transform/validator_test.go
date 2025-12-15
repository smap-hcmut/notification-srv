package transform

import (
	"testing"
)

func TestInputValidator_ValidateProjectInput(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		payload string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid project input",
			payload: `{"status": "PROCESSING", "progress": {"current": 50, "total": 100, "percentage": 50.0, "eta": 10.5, "errors": []}}`,
			wantErr: false,
		},
		{
			name:    "valid project input without progress",
			payload: `{"status": "COMPLETED"}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			payload: `{"status": "PROCESSING"`,
			wantErr: true,
			errMsg:  "invalid JSON format",
		},
		{
			name:    "missing status is valid (status is optional)",
			payload: `{"progress": {"current": 50, "total": 100}}`,
			wantErr: false,
		},
		{
			name:    "invalid status",
			payload: `{"status": "UNKNOWN"}`,
			wantErr: true,
			errMsg:  "project input validation failed",
		},
		{
			name:    "invalid progress values",
			payload: `{"status": "PROCESSING", "progress": {"current": -1, "total": 100}}`,
			wantErr: true,
			errMsg:  "project input validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateProjectInput(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateProjectInput() error = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestInputValidator_ValidateJobInput(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name    string
		payload string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid job input",
			payload: `{"platform": "TIKTOK", "status": "PROCESSING", "progress": {"current": 10, "total": 50, "percentage": 20.0, "eta": 30.0, "errors": []}}`,
			wantErr: false,
		},
		{
			name:    "valid job input with batch",
			payload: `{"platform": "YOUTUBE", "status": "PROCESSING", "batch": {"keyword": "test", "content_list": [], "crawled_at": "2024-01-01T00:00:00Z"}}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			payload: `{"platform": "TIKTOK"`,
			wantErr: true,
			errMsg:  "invalid JSON format",
		},
		{
			name:    "missing platform",
			payload: `{"status": "PROCESSING"}`,
			wantErr: true,
			errMsg:  "job input validation failed",
		},
		{
			name:    "invalid platform",
			payload: `{"platform": "FACEBOOK", "status": "PROCESSING"}`,
			wantErr: true,
			errMsg:  "job input validation failed",
		},
		{
			name:    "missing status is valid (status is optional)",
			payload: `{"platform": "TIKTOK"}`,
			wantErr: false,
		},
		{
			name:    "invalid status",
			payload: `{"platform": "TIKTOK", "status": "UNKNOWN"}`,
			wantErr: true,
			errMsg:  "job input validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJobInput(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJobInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateJobInput() error = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestValidateTopicFormat(t *testing.T) {
	tests := []struct {
		name       string
		topic      string
		wantType   string
		wantID     string
		wantUserID string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid project topic",
			topic:      "project:proj_123:user_456",
			wantType:   "project",
			wantID:     "proj_123",
			wantUserID: "user_456",
			wantErr:    false,
		},
		{
			name:       "valid job topic",
			topic:      "job:job_789:user_456",
			wantType:   "job",
			wantID:     "job_789",
			wantUserID: "user_456",
			wantErr:    false,
		},
		{
			name:    "invalid format - too few parts",
			topic:   "project:proj_123",
			wantErr: true,
			errMsg:  "invalid topic format",
		},
		{
			name:    "invalid format - too many parts",
			topic:   "project:proj_123:user_456:extra",
			wantErr: true,
			errMsg:  "invalid topic format",
		},
		{
			name:    "invalid topic type",
			topic:   "notification:proj_123:user_456",
			wantErr: true,
			errMsg:  "invalid topic type",
		},
		{
			name:    "invalid ID format - special chars",
			topic:   "project:proj@123:user_456",
			wantErr: true,
			errMsg:  "invalid project ID",
		},
		{
			name:    "invalid user ID format - too long",
			topic:   "project:proj_123:" + generateLongString(51),
			wantErr: true,
			errMsg:  "invalid user ID",
		},
		{
			name:    "empty ID",
			topic:   "project::user_456",
			wantErr: true,
			errMsg:  "invalid project ID",
		},
		{
			name:    "empty user ID",
			topic:   "project:proj_123:",
			wantErr: true,
			errMsg:  "invalid user ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topicType, id, userID, err := ValidateTopicFormat(tt.topic)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTopicFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateTopicFormat() error = %v, want to contain %v", err.Error(), tt.errMsg)
				}
				return
			}

			if topicType != tt.wantType {
				t.Errorf("ValidateTopicFormat() topicType = %v, want %v", topicType, tt.wantType)
			}
			if id != tt.wantID {
				t.Errorf("ValidateTopicFormat() id = %v, want %v", id, tt.wantID)
			}
			if userID != tt.wantUserID {
				t.Errorf("ValidateTopicFormat() userID = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}

func TestSplitTopic(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		want  []string
	}{
		{
			name:  "normal topic",
			topic: "project:proj_123:user_456",
			want:  []string{"project", "proj_123", "user_456"},
		},
		{
			name:  "single part",
			topic: "project",
			want:  []string{"project"},
		},
		{
			name:  "empty parts",
			topic: ":::",
			want:  []string{"", "", "", ""},
		},
		{
			name:  "empty topic",
			topic: "",
			want:  []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitTopic(tt.topic)
			if len(got) != len(tt.want) {
				t.Errorf("splitTopic() = %v, want %v", got, tt.want)
				return
			}
			for i, part := range got {
				if part != tt.want[i] {
					t.Errorf("splitTopic()[%d] = %v, want %v", i, part, tt.want[i])
				}
			}
		})
	}
}

func TestValidateIDFormat(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid alphanumeric",
			id:      "proj123",
			wantErr: false,
		},
		{
			name:    "valid with underscore",
			id:      "proj_123",
			wantErr: false,
		},
		{
			name:    "valid with hyphen",
			id:      "proj-123",
			wantErr: false,
		},
		{
			name:    "valid mixed case",
			id:      "Proj_123",
			wantErr: false,
		},
		{
			name:    "empty ID",
			id:      "",
			wantErr: true,
			errMsg:  "ID cannot be empty",
		},
		{
			name:    "too long ID",
			id:      generateLongString(51),
			wantErr: true,
			errMsg:  "ID too long",
		},
		{
			name:    "invalid character - space",
			id:      "proj 123",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "invalid character - special",
			id:      "proj@123",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "invalid character - dot",
			id:      "proj.123",
			wantErr: true,
			errMsg:  "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIDFormat(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIDFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateIDFormat() error = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestIsValidIDChar(t *testing.T) {
	tests := []struct {
		name string
		char rune
		want bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'A', true},
		{"digit", '5', true},
		{"underscore", '_', true},
		{"hyphen", '-', true},
		{"space", ' ', false},
		{"at symbol", '@', false},
		{"dot", '.', false},
		{"slash", '/', false},
		{"unicode", '🎉', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidIDChar(tt.char); got != tt.want {
				t.Errorf("isValidIDChar() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions for tests

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func generateLongString(length int) string {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = 'a'
	}
	return string(result)
}
