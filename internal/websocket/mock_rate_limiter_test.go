package websocket_test

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockRateLimiter implements ConnectionRateLimiter interface for testing
type MockRateLimiter struct {
	mu           sync.Mutex
	connections  map[string]int
	maxConns     int
	rateLimit    int
	window       time.Duration
	requestCount map[string]int
	lastReset    time.Time
}

func NewMockRateLimiter(maxConns int, rateLimit int, window time.Duration) *MockRateLimiter {
	return &MockRateLimiter{
		connections:  make(map[string]int),
		maxConns:     maxConns,
		rateLimit:    rateLimit,
		window:       window,
		requestCount: make(map[string]int),
		lastReset:    time.Now(),
	}
}

func (m *MockRateLimiter) CheckAndTrackConnection(ctx context.Context, userID, projectID, jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Rate limiting logic
	if time.Since(m.lastReset) > m.window {
		m.requestCount = make(map[string]int)
		m.lastReset = time.Now()
	}

	if m.requestCount[userID] >= m.rateLimit {
		return errors.New("rate limit exceeded")
	}
	m.requestCount[userID]++

	// Connection limit logic
	if m.connections[userID] >= m.maxConns {
		return errors.New("max connections exceeded")
	}
	m.connections[userID]++

	return nil
}

func (m *MockRateLimiter) UntrackConnection(userID, projectID, jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connections[userID] > 0 {
		m.connections[userID]--
	}
}
