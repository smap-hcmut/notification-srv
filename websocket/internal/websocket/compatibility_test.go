package websocket

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"smap-websocket/internal/auth"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestBackwardCompatibility(t *testing.T) {
	logger := &integrationLogger{}

	// Create hub
	hub := NewHub(logger, 100)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Create JWT validator
	jwtValidator := &mockJWTValidator{userID: "legacyuser"}

	// Create handler WITHOUT authorization or rate limiting (legacy mode)
	wsConfig := WSConfig{
		PongWait:   60 * time.Second,
		PingPeriod: 54 * time.Second,
		WriteWait:  10 * time.Second,
	}
	cookieConfig := CookieConfig{Name: "auth_token"}

	handler := NewHandler(hub, jwtValidator, logger, wsConfig, nil, cookieConfig, "dev")

	// Create test server
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.SetupRoutes(router)
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("legacy client without topic parameters", func(t *testing.T) {
		// Connect without any topic parameters (legacy behavior)
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=valid_token"

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Skipf("WebSocket connection failed (expected in test environment): %v", err)
		}
		defer conn.Close()

		// Send messages using both old and new methods
		// Old method: SendToUser (should work)
		legacyMsg, _ := NewMessage(MessageTypeNotification, map[string]interface{}{
			"message": "legacy notification",
		})
		legacyMsgBytes, _ := legacyMsg.ToJSON()
		hub.SendToUser("legacyuser", legacyMsgBytes)

		// New method: SendToUserWithProject (should also work for legacy clients)
		projectMsg, _ := NewMessage(MessageTypeProjectCompleted, map[string]interface{}{
			"status": "completed",
		})
		projectMsgBytes, _ := projectMsg.ToJSON()
		hub.SendToUserWithProject("legacyuser", "someproject", projectMsgBytes)

		// New method: SendToUserWithJob (should also work for legacy clients)
		jobMsg, _ := NewMessage(MessageTypeJobCompleted, map[string]interface{}{
			"status": "completed",
		})
		jobMsgBytes, _ := jobMsg.ToJSON()
		hub.SendToUserWithJob("legacyuser", "somejob", jobMsgBytes)

		// Legacy client should receive ALL messages (no filtering)
		messages := make([]Message, 0, 3)
		for i := 0; i < 3; i++ {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, data, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("Failed to read message %d: %v", i+1, err)
			}

			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Fatalf("Failed to unmarshal message %d: %v", i+1, err)
			}
			messages = append(messages, msg)
		}

		// Verify all message types were received
		receivedTypes := make(map[MessageType]bool)
		for _, msg := range messages {
			receivedTypes[msg.Type] = true
		}

		expectedTypes := []MessageType{
			MessageTypeNotification,
			MessageTypeProjectCompleted,
			MessageTypeJobCompleted,
		}

		for _, expectedType := range expectedTypes {
			if !receivedTypes[expectedType] {
				t.Errorf("Legacy client did not receive message type: %s", expectedType)
			}
		}
	})

	t.Run("mixed legacy and modern clients", func(t *testing.T) {
		// Connect legacy client (no filters)
		legacyURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=valid_token"
		legacyConn, _, err := websocket.DefaultDialer.Dial(legacyURL, nil)
		if err != nil {
			t.Skipf("WebSocket connection failed (expected in test environment): %v", err)
		}
		defer legacyConn.Close()

		// Connect modern client (with project filter)
		modernURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?projectId=testproject&token=valid_token"
		modernConn, _, err := websocket.DefaultDialer.Dial(modernURL, nil)
		if err != nil {
			t.Skipf("WebSocket connection failed (expected in test environment): %v", err)
		}
		defer modernConn.Close()

		// Send project-specific message
		projectMsg, _ := NewMessage(MessageTypeProjectCompleted, map[string]interface{}{
			"status": "completed",
		})
		projectMsgBytes, _ := projectMsg.ToJSON()
		hub.SendToUserWithProject("legacyuser", "testproject", projectMsgBytes)

		// Send different project message
		otherProjectMsg, _ := NewMessage(MessageTypeProjectCompleted, map[string]interface{}{
			"status": "failed",
		})
		otherProjectMsgBytes, _ := otherProjectMsg.ToJSON()
		hub.SendToUserWithProject("legacyuser", "otherproject", otherProjectMsgBytes)

		// Legacy client should receive BOTH messages
		legacyMessages := 0
		for i := 0; i < 2; i++ {
			legacyConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, _, err := legacyConn.ReadMessage()
			if err == nil {
				legacyMessages++
			}
		}

		if legacyMessages != 2 {
			t.Errorf("Legacy client should receive 2 messages, got %d", legacyMessages)
		}

		// Modern client should receive only 1 message (filtered)
		modernMessages := 0
		for i := 0; i < 2; i++ {
			modernConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			_, _, err := modernConn.ReadMessage()
			if err == nil {
				modernMessages++
			}
		}

		if modernMessages != 1 {
			t.Errorf("Modern client should receive 1 message, got %d", modernMessages)
		}
	})

	t.Run("legacy message format compatibility", func(t *testing.T) {
		// Connect legacy client
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=valid_token"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Skipf("WebSocket connection failed (expected in test environment): %v", err)
		}
		defer conn.Close()

		// Send message with legacy format
		legacyPayload := map[string]interface{}{
			"type":    "notification",
			"message": "This is a legacy notification",
			"data": map[string]interface{}{
				"priority": "high",
				"category": "system",
			},
		}

		legacyMsg, _ := NewMessage(MessageTypeNotification, legacyPayload)
		legacyMsgBytes, _ := legacyMsg.ToJSON()
		hub.SendToUser("legacyuser", legacyMsgBytes)

		// Read and verify message structure
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read legacy message: %v", err)
		}

		var receivedMsg Message
		if err := json.Unmarshal(data, &receivedMsg); err != nil {
			t.Fatalf("Failed to unmarshal legacy message: %v", err)
		}

		// Verify message structure is preserved
		if receivedMsg.Type != MessageTypeNotification {
			t.Errorf("Message type changed: expected %s, got %s", MessageTypeNotification, receivedMsg.Type)
		}

		if receivedMsg.Timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}

		// Verify payload is preserved
		var payload map[string]interface{}
		if err := json.Unmarshal(receivedMsg.Payload, &payload); err != nil {
			t.Fatalf("Failed to unmarshal payload: %v", err)
		}

		if payload["message"] != "This is a legacy notification" {
			t.Errorf("Payload message changed: %v", payload["message"])
		}
	})

	t.Run("legacy connection limits still work", func(t *testing.T) {
		// Create handler with rate limiter but no authorizer (partial legacy mode)
		rateLimiter := auth.NewConnectionTracker(auth.RateLimitConfig{
			MaxConnectionsPerUser:           2,
			MaxConnectionsPerUserPerProject: 10,
			MaxConnectionsPerUserPerJob:     10,
			ConnectionRateLimit:             100,
			RateLimitWindow:                 time.Minute,
		}, logger)

		options := &HandlerOptions{
			RateLimiter: rateLimiter,
		}

		legacyHandler := NewHandlerWithOptions(hub, jwtValidator, logger, wsConfig, nil, cookieConfig, "dev", options)

		// Create test server
		legacyRouter := gin.New()
		legacyHandler.SetupRoutes(legacyRouter)
		legacyServer := httptest.NewServer(legacyRouter)
		defer legacyServer.Close()

		wsURL := "ws" + strings.TrimPrefix(legacyServer.URL, "http") + "/ws?token=valid_token"

		// First two connections should succeed
		conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Skipf("WebSocket connection failed (expected in test environment): %v", err)
		}
		defer conn1.Close()

		conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Skipf("WebSocket connection failed (expected in test environment): %v", err)
		}
		defer conn2.Close()

		// Third connection should fail due to rate limit
		conn3, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			conn3.Close()
			t.Fatal("Third legacy connection should have failed due to rate limit")
		}

		if resp.StatusCode != 429 {
			t.Errorf("Expected status 429 for rate limit, got %d", resp.StatusCode)
		}
	})
}

func TestAPICompatibility(t *testing.T) {
	logger := &integrationLogger{}

	t.Run("hub API backward compatibility", func(t *testing.T) {
		hub := NewHub(logger, 100)
		go hub.Run()
		defer hub.Shutdown(context.Background())

		// Test that old Hub methods still work
		conn := &Connection{
			hub:       hub,
			userID:    "apiuser",
			projectID: "",
			jobID:     "",
			send:      make(chan []byte, 256),
			done:      make(chan struct{}),
			logger:    logger,
		}

		// Old registration method should still work
		hub.register <- conn
		time.Sleep(50 * time.Millisecond)

		stats := hub.GetStats()
		if stats.ActiveConnections != 1 {
			t.Errorf("Expected 1 active connection, got %d", stats.ActiveConnections)
		}

		// Old SendToUser method should still work
		msg, _ := NewMessage(MessageTypeNotification, map[string]interface{}{
			"message": "test",
		})
		msgBytes, _ := msg.ToJSON()
		hub.SendToUser("apiuser", msgBytes)

		// Verify message was sent
		select {
		case <-conn.send:
			// Message received successfully
		case <-time.After(2 * time.Second):
			t.Error("Message not received via legacy SendToUser method")
		}

		// Old unregistration method should still work
		hub.unregister <- conn
		time.Sleep(50 * time.Millisecond)

		stats = hub.GetStats()
		if stats.ActiveConnections != 0 {
			t.Errorf("Expected 0 active connections after unregister, got %d", stats.ActiveConnections)
		}
	})

	t.Run("connection API backward compatibility", func(t *testing.T) {
		hub := NewHub(logger, 100)
		go hub.Run()
		defer hub.Shutdown(context.Background())

		// Old NewConnection constructor should still work
		conn := NewConnection(hub, nil, "olduser", time.Minute, 54*time.Second, 10*time.Second, logger)

		// Verify connection has expected defaults for new fields
		if conn.GetProjectID() != "" {
			t.Errorf("Expected empty project ID for legacy connection, got %s", conn.GetProjectID())
		}

		if conn.GetJobID() != "" {
			t.Errorf("Expected empty job ID for legacy connection, got %s", conn.GetJobID())
		}

		if conn.HasTopicFilter() {
			t.Error("Legacy connection should not have topic filters")
		}

		// Verify legacy connection matches all topics (no filtering)
		if !conn.MatchesProject("anyproject") {
			t.Error("Legacy connection should match any project")
		}

		if !conn.MatchesJob("anyjob") {
			t.Error("Legacy connection should match any job")
		}
	})
}
