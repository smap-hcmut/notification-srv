package auth

import (
	"context"
	"testing"
	"time"
)

func TestConnectionTracker(t *testing.T) {
	logger := &mockLogger{}
	ctx := context.Background()

	t.Run("allows connections within limits", func(t *testing.T) {
		config := RateLimitConfig{
			MaxConnectionsPerUser:           5,
			MaxConnectionsPerUserPerProject: 2,
			MaxConnectionsPerUserPerJob:     2,
			ConnectionRateLimit:             10,
			RateLimitWindow:                 time.Minute,
		}
		tracker := NewConnectionTracker(config, logger)

		err := tracker.CheckAndTrackConnection(ctx, "user1", "proj1", "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		count := tracker.GetUserConnectionCount("user1")
		if count != 1 {
			t.Errorf("expected 1 connection, got %d", count)
		}
	})

	t.Run("enforces max connections per user", func(t *testing.T) {
		config := RateLimitConfig{
			MaxConnectionsPerUser:           2,
			MaxConnectionsPerUserPerProject: 5,
			MaxConnectionsPerUserPerJob:     5,
			ConnectionRateLimit:             100,
			RateLimitWindow:                 time.Minute,
		}
		tracker := NewConnectionTracker(config, logger)

		// First two connections should succeed
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "")
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "")

		// Third should fail
		err := tracker.CheckAndTrackConnection(ctx, "user1", "", "")
		if err == nil {
			t.Error("expected rate limit error")
		}
		if !IsRateLimitError(err) {
			t.Errorf("expected RateLimitError, got %T", err)
		}
	})

	t.Run("enforces max connections per user per project", func(t *testing.T) {
		config := RateLimitConfig{
			MaxConnectionsPerUser:           10,
			MaxConnectionsPerUserPerProject: 2,
			MaxConnectionsPerUserPerJob:     5,
			ConnectionRateLimit:             100,
			RateLimitWindow:                 time.Minute,
		}
		tracker := NewConnectionTracker(config, logger)

		// First two connections to proj1 should succeed
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "proj1", "")
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "proj1", "")

		// Third to proj1 should fail
		err := tracker.CheckAndTrackConnection(ctx, "user1", "proj1", "")
		if err == nil {
			t.Error("expected rate limit error")
		}

		// But connection to proj2 should succeed
		err = tracker.CheckAndTrackConnection(ctx, "user1", "proj2", "")
		if err != nil {
			t.Errorf("unexpected error for different project: %v", err)
		}
	})

	t.Run("enforces max connections per user per job", func(t *testing.T) {
		config := RateLimitConfig{
			MaxConnectionsPerUser:           10,
			MaxConnectionsPerUserPerProject: 5,
			MaxConnectionsPerUserPerJob:     2,
			ConnectionRateLimit:             100,
			RateLimitWindow:                 time.Minute,
		}
		tracker := NewConnectionTracker(config, logger)

		// First two connections to job1 should succeed
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "job1")
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "job1")

		// Third to job1 should fail
		err := tracker.CheckAndTrackConnection(ctx, "user1", "", "job1")
		if err == nil {
			t.Error("expected rate limit error")
		}

		// But connection to job2 should succeed
		err = tracker.CheckAndTrackConnection(ctx, "user1", "", "job2")
		if err != nil {
			t.Errorf("unexpected error for different job: %v", err)
		}
	})

	t.Run("enforces connection rate limit", func(t *testing.T) {
		config := RateLimitConfig{
			MaxConnectionsPerUser:           100,
			MaxConnectionsPerUserPerProject: 100,
			MaxConnectionsPerUserPerJob:     100,
			ConnectionRateLimit:             3,
			RateLimitWindow:                 time.Minute,
		}
		tracker := NewConnectionTracker(config, logger)

		// First three connections should succeed
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "")
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "")
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "")

		// Fourth should fail due to rate limit
		err := tracker.CheckAndTrackConnection(ctx, "user1", "", "")
		if err == nil {
			t.Error("expected rate limit error")
		}
		if !IsRateLimitError(err) {
			t.Errorf("expected RateLimitError, got %T", err)
		}
	})

	t.Run("untrack connection decrements counts", func(t *testing.T) {
		config := RateLimitConfig{
			MaxConnectionsPerUser:           2,
			MaxConnectionsPerUserPerProject: 2,
			MaxConnectionsPerUserPerJob:     2,
			ConnectionRateLimit:             100,
			RateLimitWindow:                 time.Minute,
		}
		tracker := NewConnectionTracker(config, logger)

		// Add two connections
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "proj1", "")
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "proj1", "")

		// Third should fail
		err := tracker.CheckAndTrackConnection(ctx, "user1", "proj1", "")
		if err == nil {
			t.Error("expected rate limit error")
		}

		// Untrack one connection
		tracker.UntrackConnection("user1", "proj1", "")

		// Now third should succeed
		err = tracker.CheckAndTrackConnection(ctx, "user1", "proj1", "")
		if err != nil {
			t.Errorf("unexpected error after untrack: %v", err)
		}
	})

	t.Run("get stats returns correct values", func(t *testing.T) {
		config := DefaultRateLimitConfig()
		tracker := NewConnectionTracker(config, logger)

		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "")
		_ = tracker.CheckAndTrackConnection(ctx, "user1", "", "")
		_ = tracker.CheckAndTrackConnection(ctx, "user2", "", "")

		stats := tracker.GetStats()
		if stats.TotalUsers != 2 {
			t.Errorf("expected 2 users, got %d", stats.TotalUsers)
		}
		if stats.TotalConnections != 3 {
			t.Errorf("expected 3 connections, got %d", stats.TotalConnections)
		}
	})
}

func TestRateLimitError(t *testing.T) {
	err := &RateLimitError{
		UserID:  "user1",
		Limit:   "max_connections_per_user",
		Current: 10,
		Max:     10,
	}

	expected := "rate limit exceeded for user user1: max_connections_per_user (current: 10, max: 10)"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	if !IsRateLimitError(err) {
		t.Error("IsRateLimitError should return true")
	}

	if IsRateLimitError(nil) {
		t.Error("IsRateLimitError(nil) should return false")
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.MaxConnectionsPerUser <= 0 {
		t.Error("MaxConnectionsPerUser should be positive")
	}
	if config.MaxConnectionsPerUserPerProject <= 0 {
		t.Error("MaxConnectionsPerUserPerProject should be positive")
	}
	if config.MaxConnectionsPerUserPerJob <= 0 {
		t.Error("MaxConnectionsPerUserPerJob should be positive")
	}
	if config.ConnectionRateLimit <= 0 {
		t.Error("ConnectionRateLimit should be positive")
	}
	if config.RateLimitWindow <= 0 {
		t.Error("RateLimitWindow should be positive")
	}
}
