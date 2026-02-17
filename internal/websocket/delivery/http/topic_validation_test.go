package http

import (
	"testing"
)

func TestValidateProjectID(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		wantErr   bool
	}{
		// Valid cases
		{"empty - no filter", "", false},
		{"simple alphanumeric", "proj123", false},
		{"with underscore", "proj_123", false},
		{"with hyphen", "proj-123", false},
		{"mixed", "proj_123-abc", false},
		{"uppercase", "PROJ123", false},
		{"single char", "a", false},
		{"max length 50", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},

		// Invalid cases
		{"too long 51 chars", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"with space", "proj 123", true},
		{"with special char @", "proj@123", true},
		{"with special char !", "proj!123", true},
		{"with dot", "proj.123", true},
		{"with slash", "proj/123", true},
		{"with colon", "proj:123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectID(tt.projectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectID(%q) error = %v, wantErr %v", tt.projectID, err, tt.wantErr)
			}
		})
	}
}

func TestValidateJobID(t *testing.T) {
	tests := []struct {
		name    string
		jobID   string
		wantErr bool
	}{
		// Valid cases
		{"empty - no filter", "", false},
		{"simple alphanumeric", "job123", false},
		{"with underscore", "job_123", false},
		{"with hyphen", "job-123", false},
		{"mixed", "job_123-abc", false},
		{"uppercase", "JOB123", false},
		{"single char", "a", false},
		{"max length 50", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},

		// Invalid cases
		{"too long 51 chars", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"with space", "job 123", true},
		{"with special char @", "job@123", true},
		{"with special char !", "job!123", true},
		{"with dot", "job.123", true},
		{"with slash", "job/123", true},
		{"with colon", "job:123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJobID(tt.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJobID(%q) error = %v, wantErr %v", tt.jobID, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTopicParameters(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		jobID     string
		wantErr   bool
	}{
		// Valid combinations
		{"both empty", "", "", false},
		{"only projectId", "proj123", "", false},
		{"only jobId", "", "job123", false},
		{"both set", "proj123", "job123", false},

		// Invalid combinations
		{"invalid projectId", "proj@123", "", true},
		{"invalid jobId", "", "job@123", true},
		{"both invalid", "proj@123", "job@123", true},
		{"projectId too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "", true},
		{"jobId too long", "", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTopicParameters(tt.projectID, tt.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTopicParameters(%q, %q) error = %v, wantErr %v", tt.projectID, tt.jobID, err, tt.wantErr)
			}
		})
	}
}

func TestTopicValidationError(t *testing.T) {
	err := &TopicValidationError{
		Field:   "projectId",
		Message: "must be 1-50 characters",
	}

	expected := "invalid projectId: must be 1-50 characters"
	if err.Error() != expected {
		t.Errorf("TopicValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}
