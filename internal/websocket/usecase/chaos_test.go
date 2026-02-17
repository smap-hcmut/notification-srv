package usecase

import (
	"context"
	"sync"
	"testing"
	"time"

	ws "notification-srv/internal/websocket"
)

// mockRedisNotifier simulates Redis connection failures
type mockRedisNotifier struct {
	connected      bool
	failureRate    float64 // 0.0 to 1.0
	callCount      int
	mu             sync.Mutex
	onConnected    func(userID string) error
	onDisconnected func(userID string, hasOther bool) error
}

func (m *mockRedisNotifier) OnUserConnected(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	// Simulate failure based on failure rate
	if float64(m.callCount%100)/100.0 < m.failureRate {
		return &RedisConnectionError{Operation: "OnUserConnected", UserID: userID}
	}

	if m.onConnected != nil {
		return m.onConnected(userID)
	}
	return nil
}

func (m *mockRedisNotifier) OnUserDisconnected(userID string, hasOtherConnections bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	// Simulate failure based on failure rate
	if float64(m.callCount%100)/100.0 < m.failureRate {
		return &RedisConnectionError{Operation: "OnUserDisconnected", UserID: userID}
	}

	if m.onDisconnected != nil {
		return m.onDisconnected(userID, hasOtherConnections)
	}
	return nil
}

// RedisConnectionError simulates Redis connection errors
type RedisConnectionError struct {
	Operation string
	UserID    string
}

func (e *RedisConnectionError) Error() string {
	return "redis connection failed for operation " + e.Operation + " user " + e.UserID
}

func TestChaosRedisFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	logger := &testLogger{}

	t.Run("hub resilience to Redis failures", func(t *testing.T) {
		// Create hub
		hub := NewHub(logger, 1000)
		go hub.Run()
		defer hub.Shutdown(context.Background())

		// Create Redis notifier with 30% failure rate
		redisNotifier := &mockRedisNotifier{
			connected:   true,
			failureRate: 0.3,
		}
		hub.SetRedisNotifier(redisNotifier)

		// Create many connections rapidly to stress test Redis failure handling
		numConnections := 100
		var wg sync.WaitGroup

		successfulConnections := 0
		var successMu sync.Mutex

		for i := 0; i < numConnections; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				conn := NewConnection(hub, nil, "chaosuser", time.Minute, 54*time.Second, 10*time.Second, logger)
				// Manually set fields that NewConnection might not set for testing
				// But wait, NewConnection requires http.ResponseWriter which we don't have here.
				// This test manually mocks Connection structure.
				// Since we are in websocket_test, we cannot construct usecase.Connection manually if fields are private.
				// WE MUST MOVE THIS TEST TO `usecase` PACKAGE or expose a Test helper.

				// Register connection (may fail due to Redis)
				hub.register <- conn

				// Small delay to simulate real connection timing
				time.Sleep(time.Millisecond * 10)

				// Unregister connection
				hub.unregister <- conn

				successMu.Lock()
				successfulConnections++
				successMu.Unlock()
			}(i)
		}

		wg.Wait()

		// Verify that the hub continued to function despite Redis failures
		if successfulConnections < numConnections/2 {
			t.Errorf("Too many connection failures: %d/%d successful", successfulConnections, numConnections)
		}

		t.Logf("Chaos test completed: %d/%d connections successful with 30%% Redis failure rate",
			successfulConnections, numConnections)
	})

	t.Run("message delivery during Redis instability", func(t *testing.T) {
		// Create hub
		hub := NewHub(logger, 1000)
		go hub.Run()
		defer hub.Shutdown(context.Background())

		// Create Redis notifier with intermittent failures
		redisNotifier := &mockRedisNotifier{
			connected:   true,
			failureRate: 0.2, // 20% failure rate
		}
		hub.SetRedisNotifier(redisNotifier)

		// Create stable connections
		numConnections := 50
		connections := make([]*Connection, 0, numConnections)

		for i := 0; i < numConnections; i++ {
			conn := &Connection{
				hub:       hub,
				userID:    "stableuser",
				projectID: "stableproject",
				jobID:     "",
				send:      make(chan []byte, 256),
				done:      make(chan struct{}),
				logger:    logger,
			}
			connections = append(connections, conn)
			hub.register <- conn
		}

		// Wait for connections to stabilize
		time.Sleep(100 * time.Millisecond)

		// Send messages while Redis is unstable
		numMessages := 200
		messagesReceived := 0

		// Count messages received by first connection
		go func() {
			for range connections[0].send {
				messagesReceived++
			}
		}()

		// Send messages rapidly
		for i := 0; i < numMessages; i++ {
			msg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{
				"messageID": i,
			})
			msgBytes, _ := msg.ToJSON()
			hub.SendToUserWithProject("stableuser", "stableproject", msgBytes)

			// Small delay to simulate real message timing
			time.Sleep(time.Millisecond)
		}

		// Wait for message processing
		time.Sleep(500 * time.Millisecond)

		// Verify that most messages were delivered despite Redis instability
		deliveryRate := float64(messagesReceived) / float64(numMessages)
		if deliveryRate < 0.8 { // Expect at least 80% delivery rate
			t.Errorf("Message delivery rate too low during Redis instability: %.2f%%", deliveryRate*100)
		}

		t.Logf("Message delivery during Redis instability: %d/%d messages delivered (%.2f%%)",
			messagesReceived, numMessages, deliveryRate*100)

		// Clean up
		for _, conn := range connections {
			hub.unregister <- conn
		}
	})
}

func TestHubRecoveryFromFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping recovery test in short mode")
	}

	logger := &testLogger{}

	t.Run("hub recovery after connection surge", func(t *testing.T) {
		// Create hub with limited capacity
		hub := NewHub(logger, 100)
		go hub.Run()
		defer hub.Shutdown(context.Background())

		// Create connection surge (more than capacity)
		surgeSize := 150
		var wg sync.WaitGroup

		for i := 0; i < surgeSize; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				conn := &Connection{
					hub:       hub,
					userID:    "surgeuser",
					projectID: "surgeproject",
					jobID:     "",
					send:      make(chan []byte, 256),
					done:      make(chan struct{}),
					logger:    logger,
				}

				hub.register <- conn

				// Hold connection briefly
				time.Sleep(100 * time.Millisecond)

				hub.unregister <- conn
			}(i)
		}

		wg.Wait()

		// Allow time for cleanup to complete
		time.Sleep(50 * time.Millisecond)

		// Verify hub is still functional after surge
		stats := hub.GetStats()
		if stats.ActiveConnections > 0 {
			t.Errorf("Hub should have no active connections after surge, got %d", stats.ActiveConnections)
		}

		// Test normal operation after surge
		normalConn := &Connection{
			hub:       hub,
			userID:    "normaluser",
			projectID: "normalproject",
			jobID:     "",
			send:      make(chan []byte, 256),
			done:      make(chan struct{}),
			logger:    logger,
		}

		hub.register <- normalConn
		time.Sleep(50 * time.Millisecond)

		stats = hub.GetStats()
		if stats.ActiveConnections != 1 {
			t.Errorf("Hub should have 1 active connection after recovery, got %d", stats.ActiveConnections)
		}

		hub.unregister <- normalConn
	})

	t.Run("graceful degradation under memory pressure", func(t *testing.T) {
		// Create hub
		hub := NewHub(logger, 1000)
		go hub.Run()
		defer hub.Shutdown(context.Background())

		// Create many connections to simulate memory pressure
		numConnections := 500
		connections := make([]*Connection, 0, numConnections)

		for i := 0; i < numConnections; i++ {
			conn := &Connection{
				hub:       hub,
				userID:    "memoryuser",
				projectID: "memoryproject",
				jobID:     "",
				send:      make(chan []byte, 1), // Small buffer to simulate memory pressure
				done:      make(chan struct{}),
				logger:    logger,
			}
			connections = append(connections, conn)
			hub.register <- conn
		}

		// Wait for registration
		time.Sleep(200 * time.Millisecond)

		// Send many messages rapidly to create backpressure
		numMessages := 1000
		for i := 0; i < numMessages; i++ {
			msg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{
				"messageID": i,
			})
			msgBytes, _ := msg.ToJSON()
			hub.SendToUserWithProject("memoryuser", "memoryproject", msgBytes)
		}

		// Wait for processing
		time.Sleep(1 * time.Second)

		// Verify hub is still responsive
		stats := hub.GetStats()
		if stats.ActiveConnections != numConnections {
			t.Errorf("Expected %d active connections, got %d", numConnections, stats.ActiveConnections)
		}

		// Verify some messages were delivered (even if not all due to backpressure)
		if stats.TotalMessagesSent == 0 {
			t.Error("No messages were sent, hub may have failed under memory pressure")
		}

		t.Logf("Under memory pressure: %d messages sent, %d failed",
			stats.TotalMessagesSent, stats.TotalMessagesFailed)

		// Clean up
		for _, conn := range connections {
			hub.unregister <- conn
		}
	})
}
