package websocket

import (
	"fmt"
	"regexp"
)

// Topic ID validation constants
const (
	MinTopicIDLength = 1
	MaxTopicIDLength = 50
)

// topicIDPattern matches valid topic IDs: alphanumeric, underscore, and hyphen
var topicIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// TopicValidationError represents a topic validation error
type TopicValidationError struct {
	Field   string
	Message string
}

func (e *TopicValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}

// ValidateProjectID validates a project ID parameter
// Returns nil if the projectID is empty (no filter) or valid
func ValidateProjectID(projectID string) error {
	if projectID == "" {
		return nil // Empty is valid (no filter)
	}
	return validateTopicID(projectID, "projectId")
}

// ValidateJobID validates a job ID parameter
// Returns nil if the jobID is empty (no filter) or valid
func ValidateJobID(jobID string) error {
	if jobID == "" {
		return nil // Empty is valid (no filter)
	}
	return validateTopicID(jobID, "jobId")
}

// validateTopicID validates a topic ID against format and length constraints
func validateTopicID(id string, fieldName string) error {
	if len(id) < MinTopicIDLength || len(id) > MaxTopicIDLength {
		return &TopicValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("must be %d-%d characters", MinTopicIDLength, MaxTopicIDLength),
		}
	}

	if !topicIDPattern.MatchString(id) {
		return &TopicValidationError{
			Field:   fieldName,
			Message: "only alphanumeric, underscore, and hyphen allowed",
		}
	}

	return nil
}

// ValidateTopicParameters validates both projectId and jobId parameters
func ValidateTopicParameters(projectID, jobID string) error {
	if err := ValidateProjectID(projectID); err != nil {
		return err
	}
	if err := ValidateJobID(jobID); err != nil {
		return err
	}
	return nil
}
