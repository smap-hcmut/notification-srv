package usecase

import (
	"context"
	"testing"
	"time"

	ws "notification-srv/internal/websocket"
)

// testLogger implements log.Logger for testing
type testLogger struct{}

func (m *testLogger) Debug(ctx context.Context, arg ...any)                    {}
func (m *testLogger) Debugf(ctx context.Context, template string, arg ...any)  {}
func (m *testLogger) Info(ctx context.Context, arg ...any)                     {}
func (m *testLogger) Infof(ctx context.Context, template string, arg ...any)   {}
func (m *testLogger) Warn(ctx context.Context, arg ...any)                     {}
func (m *testLogger) Warnf(ctx context.Context, template string, arg ...any)   {}
func (m *testLogger) Error(ctx context.Context, arg ...any)                    {}
func (m *testLogger) Errorf(ctx context.Context, template string, arg ...any)  {}
func (m *testLogger) Fatal(ctx context.Context, arg ...any)                    {}
func (m *testLogger) Fatalf(ctx context.Context, template string, arg ...any)  {}
func (m *testLogger) DPanic(ctx context.Context, arg ...any)                   {}
func (m *testLogger) DPanicf(ctx context.Context, template string, arg ...any) {}
func (m *testLogger) Panic(ctx context.Context, arg ...any)                    {}
func (m *testLogger) Panicf(ctx context.Context, template string, arg ...any)  {}

func TestHubSendToUserWithProject(t *testing.T) {
	logger := &testLogger{}
	hub := NewHub(logger, 100)

	// Start hub in background
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Create test connections
	userID := "user123"
	projectID := "proj123"

	// Connection with project filter
	conn1 := &Connection{
		hub:       hub,
		userID:    userID,
		projectID: projectID,
		jobID:     "",
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger,
	}

	// Connection without filter (should receive all)
	conn2 := &Connection{
		hub:       hub,
		userID:    userID,
		projectID: "",
		jobID:     "",
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger,
	}

	// Connection with different project filter
	conn3 := &Connection{
		hub:       hub,
		userID:    userID,
		projectID: "proj456",
		jobID:     "",
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger,
	}

	// Register connections
	hub.register <- conn1
	hub.register <- conn2
	hub.register <- conn3

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Send message to project
	msg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{"status": "completed"})
	msgBytes, _ := msg.ToJSON()
	hub.SendToUserWithProject(userID, projectID, msgBytes)

	// Wait for message delivery
	time.Sleep(50 * time.Millisecond)

	// Check conn1 received message (has matching project filter)
	select {
	case <-conn1.send:
		// Expected
	default:
		t.Error("conn1 (matching project filter) should have received message")
	}

	// Check conn2 received message (no filter, receives all)
	select {
	case <-conn2.send:
		// Expected
	default:
		t.Error("conn2 (no filter) should have received message")
	}

	// Check conn3 did NOT receive message (different project filter)
	select {
	case <-conn3.send:
		t.Error("conn3 (different project filter) should NOT have received message")
	default:
		// Expected
	}
}

func TestHubSendToUserWithJob(t *testing.T) {
	logger := &testLogger{}
	hub := NewHub(logger, 100)

	// Start hub in background
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Create test connections
	userID := "user123"
	jobID := "job123"

	// Connection with job filter
	conn1 := &Connection{
		hub:       hub,
		userID:    userID,
		projectID: "",
		jobID:     jobID,
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger,
	}

	// Connection without filter (should receive all)
	conn2 := &Connection{
		hub:       hub,
		userID:    userID,
		projectID: "",
		jobID:     "",
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger,
	}

	// Connection with different job filter
	conn3 := &Connection{
		hub:       hub,
		userID:    userID,
		projectID: "",
		jobID:     "job456",
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger,
	}

	// Register connections
	hub.register <- conn1
	hub.register <- conn2
	hub.register <- conn3

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Send message to job
	msg, _ := ws.NewMessage(ws.MessageTypeJobCompleted, map[string]interface{}{"status": "completed"})
	msgBytes, _ := msg.ToJSON()
	hub.SendToUserWithJob(userID, jobID, msgBytes)

	// Wait for message delivery
	time.Sleep(50 * time.Millisecond)

	// Check conn1 received message (has matching job filter)
	select {
	case <-conn1.send:
		// Expected
	default:
		t.Error("conn1 (matching job filter) should have received message")
	}

	// Check conn2 received message (no filter, receives all)
	select {
	case <-conn2.send:
		// Expected
	default:
		t.Error("conn2 (no filter) should have received message")
	}

	// Check conn3 did NOT receive message (different job filter)
	select {
	case <-conn3.send:
		t.Error("conn3 (different job filter) should NOT have received message")
	default:
		// Expected
	}
}

func TestHubFilteredConnectionsMetrics(t *testing.T) {
	logger := &testLogger{}
	hub := NewHub(logger, 100)

	// Start hub in background
	go hub.Run()
	defer hub.Shutdown(context.Background())

	userID := "user123"

	// Connection with filter
	conn1 := &Connection{
		hub:       hub,
		userID:    userID,
		projectID: "proj123",
		jobID:     "",
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger,
	}

	// Connection without filter
	conn2 := &Connection{
		hub:       hub,
		userID:    userID,
		projectID: "",
		jobID:     "",
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger,
	}

	// Register connections
	hub.register <- conn1
	hub.register <- conn2

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	stats := hub.GetStats()
	if stats.ActiveConnections != 2 {
		t.Errorf("ActiveConnections = %d, want 2", stats.ActiveConnections)
	}
	if stats.FilteredConnections != 1 {
		t.Errorf("FilteredConnections = %d, want 1", stats.FilteredConnections)
	}

	// Unregister filtered connection
	hub.unregister <- conn1
	time.Sleep(50 * time.Millisecond)

	stats = hub.GetStats()
	if stats.FilteredConnections != 0 {
		t.Errorf("FilteredConnections after unregister = %d, want 0", stats.FilteredConnections)
	}
}

func TestHubSendToUserWithProjectNoConnections(t *testing.T) {
	logger := &testLogger{}
	hub := NewHub(logger, 100)

	// Start hub in background
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Send message to non-existent user - should not panic
	msg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{"status": "completed"})
	msgBytes, _ := msg.ToJSON()
	hub.SendToUserWithProject("nonexistent", "proj123", msgBytes)

	// If we get here without panic, test passes
}

func TestHubSendToUserWithJobNoConnections(t *testing.T) {
	logger := &testLogger{}
	hub := NewHub(logger, 100)

	// Start hub in background
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Send message to non-existent user - should not panic
	msg, _ := ws.NewMessage(ws.MessageTypeJobCompleted, map[string]interface{}{"status": "completed"})
	msgBytes, _ := msg.ToJSON()
	hub.SendToUserWithJob("nonexistent", "job123", msgBytes)

	// If we get here without panic, test passes
}
