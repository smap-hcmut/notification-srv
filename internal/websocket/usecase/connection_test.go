package usecase

import (
	"context"
	"testing"
	"time"
)

// mockLogger implements log.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Debugf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Info(ctx context.Context, arg ...any)                     {}
func (m *mockLogger) Infof(ctx context.Context, template string, arg ...any)   {}
func (m *mockLogger) Warn(ctx context.Context, arg ...any)                     {}
func (m *mockLogger) Warnf(ctx context.Context, template string, arg ...any)   {}
func (m *mockLogger) Error(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Errorf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Fatal(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Fatalf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) DPanic(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) DPanicf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Panic(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Panicf(ctx context.Context, template string, arg ...any)  {}

func TestConnectionTopicFilters(t *testing.T) {
	logger := &mockLogger{}

	t.Run("NewConnection creates connection without filters", func(t *testing.T) {
		conn := NewConnection(nil, nil, "user123", time.Second, time.Second, time.Second, logger)

		if conn.GetProjectID() != "" {
			t.Errorf("GetProjectID() = %q, want empty", conn.GetProjectID())
		}
		if conn.GetJobID() != "" {
			t.Errorf("GetJobID() = %q, want empty", conn.GetJobID())
		}
		if conn.HasProjectFilter() {
			t.Error("HasProjectFilter() = true, want false")
		}
		if conn.HasJobFilter() {
			t.Error("HasJobFilter() = true, want false")
		}
		if conn.HasTopicFilter() {
			t.Error("HasTopicFilter() = true, want false")
		}
	})

	t.Run("NewConnectionWithFilters creates connection with project filter", func(t *testing.T) {
		conn := NewConnectionWithFilters(nil, nil, "user123", "proj123", "", time.Second, time.Second, time.Second, logger)

		if conn.GetProjectID() != "proj123" {
			t.Errorf("GetProjectID() = %q, want %q", conn.GetProjectID(), "proj123")
		}
		if conn.GetJobID() != "" {
			t.Errorf("GetJobID() = %q, want empty", conn.GetJobID())
		}
		if !conn.HasProjectFilter() {
			t.Error("HasProjectFilter() = false, want true")
		}
		if conn.HasJobFilter() {
			t.Error("HasJobFilter() = true, want false")
		}
		if !conn.HasTopicFilter() {
			t.Error("HasTopicFilter() = false, want true")
		}
	})

	t.Run("NewConnectionWithFilters creates connection with job filter", func(t *testing.T) {
		conn := NewConnectionWithFilters(nil, nil, "user123", "", "job456", time.Second, time.Second, time.Second, logger)

		if conn.GetProjectID() != "" {
			t.Errorf("GetProjectID() = %q, want empty", conn.GetProjectID())
		}
		if conn.GetJobID() != "job456" {
			t.Errorf("GetJobID() = %q, want %q", conn.GetJobID(), "job456")
		}
		if conn.HasProjectFilter() {
			t.Error("HasProjectFilter() = true, want false")
		}
		if !conn.HasJobFilter() {
			t.Error("HasJobFilter() = false, want true")
		}
		if !conn.HasTopicFilter() {
			t.Error("HasTopicFilter() = false, want true")
		}
	})

	t.Run("NewConnectionWithFilters creates connection with both filters", func(t *testing.T) {
		conn := NewConnectionWithFilters(nil, nil, "user123", "proj123", "job456", time.Second, time.Second, time.Second, logger)

		if conn.GetProjectID() != "proj123" {
			t.Errorf("GetProjectID() = %q, want %q", conn.GetProjectID(), "proj123")
		}
		if conn.GetJobID() != "job456" {
			t.Errorf("GetJobID() = %q, want %q", conn.GetJobID(), "job456")
		}
		if !conn.HasProjectFilter() {
			t.Error("HasProjectFilter() = false, want true")
		}
		if !conn.HasJobFilter() {
			t.Error("HasJobFilter() = false, want true")
		}
		if !conn.HasTopicFilter() {
			t.Error("HasTopicFilter() = false, want true")
		}
	})
}

func TestConnectionMatchesProject(t *testing.T) {
	logger := &mockLogger{}

	tests := []struct {
		name            string
		connProjectID   string
		targetProjectID string
		want            bool
	}{
		{"no filter matches any project", "", "proj123", true},
		{"no filter matches empty project", "", "", true},
		{"filter matches same project", "proj123", "proj123", true},
		{"filter does not match different project", "proj123", "proj456", false},
		{"filter does not match empty project", "proj123", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := NewConnectionWithFilters(nil, nil, "user123", tt.connProjectID, "", time.Second, time.Second, time.Second, logger)
			got := conn.MatchesProject(tt.targetProjectID)
			if got != tt.want {
				t.Errorf("MatchesProject(%q) = %v, want %v", tt.targetProjectID, got, tt.want)
			}
		})
	}
}

func TestConnectionMatchesJob(t *testing.T) {
	logger := &mockLogger{}

	tests := []struct {
		name        string
		connJobID   string
		targetJobID string
		want        bool
	}{
		{"no filter matches any job", "", "job123", true},
		{"no filter matches empty job", "", "", true},
		{"filter matches same job", "job123", "job123", true},
		{"filter does not match different job", "job123", "job456", false},
		{"filter does not match empty job", "job123", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := NewConnectionWithFilters(nil, nil, "user123", "", tt.connJobID, time.Second, time.Second, time.Second, logger)
			got := conn.MatchesJob(tt.targetJobID)
			if got != tt.want {
				t.Errorf("MatchesJob(%q) = %v, want %v", tt.targetJobID, got, tt.want)
			}
		})
	}
}

func TestConnectionGetUserID(t *testing.T) {
	logger := &mockLogger{}

	conn := NewConnection(nil, nil, "user123", time.Second, time.Second, time.Second, logger)
	if conn.GetUserID() != "user123" {
		t.Errorf("GetUserID() = %q, want %q", conn.GetUserID(), "user123")
	}
}
