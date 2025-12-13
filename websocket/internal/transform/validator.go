package transform

import (
	"encoding/json"
	"fmt"

	"smap-websocket/internal/types"
)

// InputValidator implements MessageValidator interface
type InputValidator struct{}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{}
}

// ValidateProjectInput validates project input message structure
func (v *InputValidator) ValidateProjectInput(payload string) error {
	// Check if JSON is valid
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("invalid JSON format")
	}
	
	// Parse into project input structure
	var projectInput types.ProjectInputMessage
	if err := json.Unmarshal([]byte(payload), &projectInput); err != nil {
		return fmt.Errorf("failed to unmarshal project input: %w", err)
	}
	
	// Validate using built-in validation
	if err := projectInput.Validate(); err != nil {
		return fmt.Errorf("project input validation failed: %w", err)
	}
	
	return nil
}

// ValidateJobInput validates job input message structure
func (v *InputValidator) ValidateJobInput(payload string) error {
	// Check if JSON is valid
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("invalid JSON format")
	}
	
	// Parse into job input structure
	var jobInput types.JobInputMessage
	if err := json.Unmarshal([]byte(payload), &jobInput); err != nil {
		return fmt.Errorf("failed to unmarshal job input: %w", err)
	}
	
	// Validate using built-in validation
	if err := jobInput.Validate(); err != nil {
		return fmt.Errorf("job input validation failed: %w", err)
	}
	
	return nil
}

// ValidateTopicFormat validates topic format and extracts components
func ValidateTopicFormat(topic string) (topicType, id, userID string, err error) {
	// Split topic into parts
	parts := splitTopic(topic)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid topic format: expected 'type:id:userID', got '%s'", topic)
	}
	
	topicType = parts[0]
	id = parts[1]  
	userID = parts[2]
	
	// Validate topic type
	if topicType != "project" && topicType != "job" {
		return "", "", "", fmt.Errorf("invalid topic type: %s", topicType)
	}
	
	// Validate ID format
	if err := validateIDFormat(id); err != nil {
		return "", "", "", fmt.Errorf("invalid %s ID: %w", topicType, err)
	}
	
	// Validate user ID format
	if err := validateIDFormat(userID); err != nil {
		return "", "", "", fmt.Errorf("invalid user ID: %w", err)
	}
	
	return topicType, id, userID, nil
}

// splitTopic splits topic string by colon delimiter
func splitTopic(topic string) []string {
	if topic == "" {
		return []string{""}
	}
	
	parts := make([]string, 0, 3)
	current := ""
	
	for _, char := range topic {
		if char == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	
	// Add the last part
	parts = append(parts, current)
	
	return parts
}

// validateIDFormat validates ID format (alphanumeric, underscore, hyphen, 1-50 chars)
func validateIDFormat(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("ID cannot be empty")
	}
	
	if len(id) > 50 {
		return fmt.Errorf("ID too long: %d characters (max 50)", len(id))
	}
	
	for _, char := range id {
		if !isValidIDChar(char) {
			return fmt.Errorf("invalid character in ID: %c (only alphanumeric, underscore, and hyphen allowed)", char)
		}
	}
	
	return nil
}

// isValidIDChar checks if character is valid for ID
func isValidIDChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		   (char >= 'A' && char <= 'Z') ||
		   (char >= '0' && char <= '9') ||
		   char == '_' ||
		   char == '-'
}