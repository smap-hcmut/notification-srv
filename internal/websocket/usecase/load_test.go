package usecase

import (
	"context"
	"fmt"
	ws "notification-srv/internal/websocket"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestLoadTopicMessageFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	logger := &testLogger{}

	// Create hub with higher limits for load testing
	hub := NewHub(logger, 10000)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Test parameters
	numUsers := 100
	connectionsPerUser := 5
	messagesPerUser := 100
	numProjects := 10

	t.Run("high concurrency message filtering", func(t *testing.T) {
		var wg sync.WaitGroup
		startTime := time.Now()

		// Create connections for each user
		connections := make(map[string][]*Connection)
		for userID := 0; userID < numUsers; userID++ {
			userIDStr := fmt.Sprintf("user%d", userID)
			connections[userIDStr] = make([]*Connection, 0, connectionsPerUser)

			for connID := 0; connID < connectionsPerUser; connID++ {
				projectID := fmt.Sprintf("proj%d", connID%numProjects)

				conn := &Connection{
					hub:       hub,
					userID:    userIDStr,
					projectID: projectID,
					jobID:     "",
					send:      make(chan []byte, 256),
					done:      make(chan struct{}),
					logger:    logger,
				}

				connections[userIDStr] = append(connections[userIDStr], conn)
				hub.register <- conn
			}
		}

		// Wait for all connections to be registered
		time.Sleep(100 * time.Millisecond)

		// Send messages concurrently
		wg.Add(numUsers)
		for userID := 0; userID < numUsers; userID++ {
			go func(uid int) {
				defer wg.Done()
				userIDStr := fmt.Sprintf("user%d", uid)

				for msgID := 0; msgID < messagesPerUser; msgID++ {
					projectID := fmt.Sprintf("proj%d", msgID%numProjects)
					msg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{
						"messageID": msgID,
						"userID":    userIDStr,
					})
					msgBytes, _ := msg.ToJSON()
					hub.SendToUserWithProject(userIDStr, projectID, msgBytes)
				}
			}(userID)
		}

		wg.Wait()

		duration := time.Since(startTime)
		totalMessages := numUsers * messagesPerUser
		messagesPerSecond := float64(totalMessages) / duration.Seconds()

		t.Logf("Load test completed:")
		t.Logf("  Users: %d", numUsers)
		t.Logf("  Connections per user: %d", connectionsPerUser)
		t.Logf("  Total connections: %d", numUsers*connectionsPerUser)
		t.Logf("  Messages per user: %d", messagesPerUser)
		t.Logf("  Total messages: %d", totalMessages)
		t.Logf("  Duration: %v", duration)
		t.Logf("  Messages/second: %.2f", messagesPerSecond)

		// Verify performance requirements
		if messagesPerSecond < 1000 {
			t.Errorf("Performance requirement not met: %.2f messages/second < 1000", messagesPerSecond)
		}

		// Clean up connections
		for _, userConns := range connections {
			for _, conn := range userConns {
				hub.unregister <- conn
			}
		}
	})
}

func TestMemoryUsageWithTopicFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	logger := &testLogger{}

	// Measure baseline memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create hub
	hub := NewHub(logger, 10000)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Create many connections with topic filters
	numConnections := 1000
	connections := make([]*Connection, 0, numConnections)

	for i := 0; i < numConnections; i++ {
		conn := &Connection{
			hub:       hub,
			userID:    fmt.Sprintf("user%d", i%100), // 100 unique users
			projectID: fmt.Sprintf("proj%d", i%50),  // 50 unique projects
			jobID:     fmt.Sprintf("job%d", i%25),   // 25 unique jobs
			send:      make(chan []byte, 256),
			done:      make(chan struct{}),
			logger:    logger,
		}
		connections = append(connections, conn)
		hub.register <- conn
	}

	// Wait for all connections to be registered
	time.Sleep(100 * time.Millisecond)

	// Measure memory after creating connections
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	memoryIncrease := m2.Alloc - m1.Alloc
	memoryPerConnection := float64(memoryIncrease) / float64(numConnections)

	t.Logf("Memory usage analysis:")
	t.Logf("  Connections: %d", numConnections)
	t.Logf("  Memory increase: %d bytes", memoryIncrease)
	t.Logf("  Memory per connection: %.2f bytes", memoryPerConnection)

	// Verify memory requirement (should be reasonable)
	// WebSocket connections with topic filters require more memory due to:
	// - gorilla/websocket connection overhead
	// - topic filter strings
	// - connection metadata
	// - Go runtime overhead
	maxMemoryPerConnection := 10240.0 // 10KB per connection is reasonable for WebSocket with filters
	if memoryPerConnection > maxMemoryPerConnection {
		t.Errorf("Memory usage too high: %.2f bytes per connection > %.2f", memoryPerConnection, maxMemoryPerConnection)
	}

	// Clean up
	for _, conn := range connections {
		hub.unregister <- conn
	}
}

func TestMessageFilteringLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}

	logger := &testLogger{}

	// Create hub
	hub := NewHub(logger, 1000)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Create connections with different filters
	numConnections := 100
	connections := make([]*Connection, 0, numConnections)

	for i := 0; i < numConnections; i++ {
		conn := &Connection{
			hub:       hub,
			userID:    "testuser",
			projectID: fmt.Sprintf("proj%d", i%10), // 10 different projects
			jobID:     "",
			send:      make(chan []byte, 256),
			done:      make(chan struct{}),
			logger:    logger,
		}
		connections = append(connections, conn)
		hub.register <- conn
	}

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Measure message filtering latency
	numMessages := 1000
	latencies := make([]time.Duration, 0, numMessages)

	for i := 0; i < numMessages; i++ {
		projectID := fmt.Sprintf("proj%d", i%10)
		msg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{
			"messageID": i,
		})
		msgBytes, _ := msg.ToJSON()

		start := time.Now()
		hub.SendToUserWithProject("testuser", projectID, msgBytes)
		latency := time.Since(start)
		latencies = append(latencies, latency)
	}

	// Calculate statistics
	var totalLatency time.Duration
	var maxLatency time.Duration
	for _, lat := range latencies {
		totalLatency += lat
		if lat > maxLatency {
			maxLatency = lat
		}
	}
	avgLatency := totalLatency / time.Duration(len(latencies))

	// Calculate 99th percentile
	// Sort latencies (simple bubble sort for small dataset)
	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}
	p99Index := int(float64(len(latencies)) * 0.99)
	p99Latency := latencies[p99Index]

	t.Logf("Message filtering latency analysis:")
	t.Logf("  Messages: %d", numMessages)
	t.Logf("  Connections: %d", numConnections)
	t.Logf("  Average latency: %v", avgLatency)
	t.Logf("  Max latency: %v", maxLatency)
	t.Logf("  99th percentile: %v", p99Latency)

	// Verify latency requirements (< 5ms for 99% of messages)
	maxAllowedLatency := 5 * time.Millisecond
	if p99Latency > maxAllowedLatency {
		t.Errorf("Latency requirement not met: 99th percentile %v > %v", p99Latency, maxAllowedLatency)
	}

	// Clean up
	for _, conn := range connections {
		hub.unregister <- conn
	}
}

func BenchmarkTopicMessageRouting(b *testing.B) {
	logger := &testLogger{}

	// Create hub
	hub := NewHub(logger, 1000)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Create test connections
	numConnections := 100
	for i := 0; i < numConnections; i++ {
		conn := &Connection{
			hub:       hub,
			userID:    fmt.Sprintf("user%d", i%10), // 10 users
			projectID: fmt.Sprintf("proj%d", i%5),  // 5 projects per user
			jobID:     "",
			send:      make(chan []byte, 256),
			done:      make(chan struct{}),
			logger:    logger,
		}
		hub.register <- conn
	}

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Benchmark message routing
	msg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{
		"status": "completed",
	})
	msgBytes, _ := msg.ToJSON()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user%d", i%10)
			projectID := fmt.Sprintf("proj%d", i%5)
			hub.SendToUserWithProject(userID, projectID, msgBytes)
			i++
		}
	})
}
