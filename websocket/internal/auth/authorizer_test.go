package auth

import (
	"context"
	"testing"
	"time"
)

// mockLogger implements log.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Debugf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Info(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Infof(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Warn(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Warnf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Error(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Errorf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Fatal(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Fatalf(ctx context.Context, template string, arg ...any) {}

// mockAuthorizer is a mock implementation for testing
type mockAuthorizer struct {
	projectAccess map[string]map[string]bool // userID -> projectID -> allowed
	jobAccess     map[string]map[string]bool // userID -> jobID -> allowed
	callCount     int
}

func newMockAuthorizer() *mockAuthorizer {
	return &mockAuthorizer{
		projectAccess: make(map[string]map[string]bool),
		jobAccess:     make(map[string]map[string]bool),
	}
}

func (m *mockAuthorizer) setProjectAccess(userID, projectID string, allowed bool) {
	if m.projectAccess[userID] == nil {
		m.projectAccess[userID] = make(map[string]bool)
	}
	m.projectAccess[userID][projectID] = allowed
}

func (m *mockAuthorizer) setJobAccess(userID, jobID string, allowed bool) {
	if m.jobAccess[userID] == nil {
		m.jobAccess[userID] = make(map[string]bool)
	}
	m.jobAccess[userID][jobID] = allowed
}

func (m *mockAuthorizer) CanAccessProject(ctx context.Context, userID, projectID string) (bool, error) {
	m.callCount++
	if m.projectAccess[userID] == nil {
		return false, nil
	}
	return m.projectAccess[userID][projectID], nil
}

func (m *mockAuthorizer) CanAccessJob(ctx context.Context, userID, jobID string) (bool, error) {
	m.callCount++
	if m.jobAccess[userID] == nil {
		return false, nil
	}
	return m.jobAccess[userID][jobID], nil
}

func TestPermissiveAuthorizer(t *testing.T) {
	logger := &mockLogger{}
	auth := NewPermissiveAuthorizer(logger)
	ctx := context.Background()

	t.Run("always allows project access", func(t *testing.T) {
		allowed, err := auth.CanAccessProject(ctx, "user1", "proj1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected access to be allowed")
		}
	})

	t.Run("always allows job access", func(t *testing.T) {
		allowed, err := auth.CanAccessJob(ctx, "user1", "job1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected access to be allowed")
		}
	})
}

func TestCachedAuthorizer(t *testing.T) {
	logger := &mockLogger{}
	ctx := context.Background()

	t.Run("caches project access results", func(t *testing.T) {
		mock := newMockAuthorizer()
		mock.setProjectAccess("user1", "proj1", true)

		cached := NewCachedAuthorizer(mock, time.Minute, logger)

		// First call should hit the delegate
		allowed, err := cached.CanAccessProject(ctx, "user1", "proj1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected access to be allowed")
		}
		if mock.callCount != 1 {
			t.Errorf("expected 1 delegate call, got %d", mock.callCount)
		}

		// Second call should use cache
		allowed, err = cached.CanAccessProject(ctx, "user1", "proj1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected access to be allowed")
		}
		if mock.callCount != 1 {
			t.Errorf("expected 1 delegate call (cached), got %d", mock.callCount)
		}
	})

	t.Run("caches job access results", func(t *testing.T) {
		mock := newMockAuthorizer()
		mock.setJobAccess("user1", "job1", true)

		cached := NewCachedAuthorizer(mock, time.Minute, logger)

		// First call should hit the delegate
		allowed, err := cached.CanAccessJob(ctx, "user1", "job1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected access to be allowed")
		}
		if mock.callCount != 1 {
			t.Errorf("expected 1 delegate call, got %d", mock.callCount)
		}

		// Second call should use cache
		allowed, err = cached.CanAccessJob(ctx, "user1", "job1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected access to be allowed")
		}
		if mock.callCount != 1 {
			t.Errorf("expected 1 delegate call (cached), got %d", mock.callCount)
		}
	})

	t.Run("cache expires", func(t *testing.T) {
		mock := newMockAuthorizer()
		mock.setProjectAccess("user1", "proj1", true)

		// Very short TTL for testing
		cached := NewCachedAuthorizer(mock, 10*time.Millisecond, logger)

		// First call
		_, _ = cached.CanAccessProject(ctx, "user1", "proj1")
		if mock.callCount != 1 {
			t.Errorf("expected 1 delegate call, got %d", mock.callCount)
		}

		// Wait for cache to expire
		time.Sleep(20 * time.Millisecond)

		// Should hit delegate again
		_, _ = cached.CanAccessProject(ctx, "user1", "proj1")
		if mock.callCount != 2 {
			t.Errorf("expected 2 delegate calls after expiry, got %d", mock.callCount)
		}
	})

	t.Run("invalidate user clears cache", func(t *testing.T) {
		mock := newMockAuthorizer()
		mock.setProjectAccess("user1", "proj1", true)

		cached := NewCachedAuthorizer(mock, time.Minute, logger)

		// Populate cache
		_, _ = cached.CanAccessProject(ctx, "user1", "proj1")
		if mock.callCount != 1 {
			t.Errorf("expected 1 delegate call, got %d", mock.callCount)
		}

		// Invalidate user
		cached.InvalidateUser("user1")

		// Should hit delegate again
		_, _ = cached.CanAccessProject(ctx, "user1", "proj1")
		if mock.callCount != 2 {
			t.Errorf("expected 2 delegate calls after invalidation, got %d", mock.callCount)
		}
	})
}

func TestAuthorizationError(t *testing.T) {
	err := &AuthorizationError{
		UserID:     "user1",
		ResourceID: "proj1",
		Resource:   "project",
		Reason:     "not a member",
	}

	expected := "unauthorized access to project proj1 for user user1: not a member"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	if !IsAuthorizationError(err) {
		t.Error("IsAuthorizationError should return true")
	}

	if IsAuthorizationError(nil) {
		t.Error("IsAuthorizationError(nil) should return false")
	}
}
