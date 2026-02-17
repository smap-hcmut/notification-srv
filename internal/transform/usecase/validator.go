package usecase

import (
	"encoding/json"
	"fmt"

	"notification-srv/internal/transform"
)

// ValidateProjectInput validates project input message structure
// Supports both legacy and phase-based formats
func (v *inputValidator) ValidateProjectInput(payload string) error {
	// Check if JSON is valid
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("invalid JSON format")
	}

	// Check if this is phase-based format
	if transform.IsPhaseBasedMessage([]byte(payload)) {
		return v.ValidateProjectPhaseInput(payload)
	}

	// Parse into legacy project input structure
	var projectInput transform.ProjectInputMessage
	if err := json.Unmarshal([]byte(payload), &projectInput); err != nil {
		return fmt.Errorf("failed to unmarshal project input: %w", err)
	}

	// Validate using built-in validation
	if err := projectInput.Validate(); err != nil {
		return fmt.Errorf("project input validation failed: %w", err)
	}

	return nil
}

// ValidateProjectPhaseInput validates phase-based project input message structure
func (v *inputValidator) ValidateProjectPhaseInput(payload string) error {
	// Check if JSON is valid
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("invalid JSON format")
	}

	// Parse into phase-based project input structure
	var phaseInput transform.ProjectPhaseInputMessage
	if err := json.Unmarshal([]byte(payload), &phaseInput); err != nil {
		return fmt.Errorf("failed to unmarshal phase-based project input: %w", err)
	}

	// Validate using built-in validation
	if err := phaseInput.Validate(); err != nil {
		return fmt.Errorf("phase-based project input validation failed: %w", err)
	}

	return nil
}

// ValidateJobInput validates job input message structure
func (v *inputValidator) ValidateJobInput(payload string) error {
	// Check if JSON is valid
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("invalid JSON format")
	}

	// Parse into job input structure
	var jobInput transform.JobInputMessage
	if err := json.Unmarshal([]byte(payload), &jobInput); err != nil {
		return fmt.Errorf("failed to unmarshal job input: %w", err)
	}

	// Validate using built-in validation
	if err := jobInput.Validate(); err != nil {
		return fmt.Errorf("job input validation failed: %w", err)
	}

	return nil
}
