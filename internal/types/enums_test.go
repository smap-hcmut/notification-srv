package types

import "testing"

func TestIsValidProjectStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"PROCESSING is valid", "PROCESSING", true},
		{"COMPLETED is valid", "COMPLETED", true},
		{"FAILED is valid", "FAILED", true},
		{"PAUSED is valid", "PAUSED", true},
		{"lowercase processing is invalid", "processing", false},
		{"UNKNOWN is invalid", "UNKNOWN", false},
		{"empty string is invalid", "", false},
		{"RUNNING is invalid", "RUNNING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidProjectStatus(tt.status)
			if result != tt.expected {
				t.Errorf("IsValidProjectStatus(%q) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestIsValidJobStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"PROCESSING is valid", "PROCESSING", true},
		{"COMPLETED is valid", "COMPLETED", true},
		{"FAILED is valid", "FAILED", true},
		{"PAUSED is valid", "PAUSED", true},
		{"lowercase processing is invalid", "processing", false},
		{"UNKNOWN is invalid", "UNKNOWN", false},
		{"empty string is invalid", "", false},
		{"RUNNING is invalid", "RUNNING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidJobStatus(tt.status)
			if result != tt.expected {
				t.Errorf("IsValidJobStatus(%q) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestIsValidPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected bool
	}{
		{"TIKTOK is valid", "TIKTOK", true},
		{"YOUTUBE is valid", "YOUTUBE", true},
		{"INSTAGRAM is valid", "INSTAGRAM", true},
		{"lowercase tiktok is invalid", "tiktok", false},
		{"FACEBOOK is invalid", "FACEBOOK", false},
		{"empty string is invalid", "", false},
		{"TWITTER is invalid", "TWITTER", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPlatform(tt.platform)
			if result != tt.expected {
				t.Errorf("IsValidPlatform(%q) = %v, want %v", tt.platform, result, tt.expected)
			}
		})
	}
}