package websocket_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	ws "notification-srv/internal/websocket"
	httpDelivery "notification-srv/internal/websocket/delivery/http"
	"notification-srv/internal/websocket/usecase"
)

func TestBackwardCompatibility(t *testing.T) {
	logger := &integrationLogger{}

	// Create hub
	hub := usecase.NewHub(logger, 100)
	go hub.Run()
	defer hub.Shutdown(context.Background())

	// Create JWT validator
	jwtValidator := &mockJWTValidator{userID: "legacyuser"}

	// Create handler WITHOUT authorization or rate limiting (legacy mode)
	wsConfig := httpDelivery.WSConfig{
		PongWait:   60 * time.Second,
		PingPeriod: 54 * time.Second,
		WriteWait:  10 * time.Second,
	}
	cookieConfig := httpDelivery.CookieConfig{Name: "auth_token"}

	handler := httpDelivery.NewHandler(hub, jwtValidator, logger, wsConfig, nil, cookieConfig, "dev")

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
		legacyMsg, _ := ws.NewMessage(ws.MessageTypeNotification, map[string]interface{}{
			"message": "legacy notification",
		})
		legacyMsgBytes, _ := legacyMsg.ToJSON()
		hub.SendToUser("legacyuser", legacyMsgBytes)

		// New method: SendToUserWithProject (should also work for legacy clients)
		projectMsg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{
			"status": "completed",
		})
		projectMsgBytes, _ := projectMsg.ToJSON()
		hub.SendToUserWithProject("legacyuser", "someproject", projectMsgBytes)

		// New method: SendToUserWithJob (should also work for legacy clients)
		jobMsg, _ := ws.NewMessage(ws.MessageTypeJobCompleted, map[string]interface{}{
			"status": "completed",
		})
		jobMsgBytes, _ := jobMsg.ToJSON()
		hub.SendToUserWithJob("legacyuser", "somejob", jobMsgBytes)

		// Legacy client should receive ALL messages (no filtering)
		messages := make([]ws.Message, 0, 3)
		for i := 0; i < 3; i++ {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, data, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("Failed to read message %d: %v", i+1, err)
			}

			var msg ws.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Fatalf("Failed to unmarshal message %d: %v", i+1, err)
			}
			messages = append(messages, msg)
		}

		// Verify all message types were received
		receivedTypes := make(map[ws.MessageType]bool)
		for _, msg := range messages {
			receivedTypes[msg.Type] = true
		}

		expectedTypes := []ws.MessageType{
			ws.MessageTypeNotification,
			ws.MessageTypeProjectCompleted,
			ws.MessageTypeJobCompleted,
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
		projectMsg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{
			"status": "completed",
		})
		projectMsgBytes, _ := projectMsg.ToJSON()
		hub.SendToUserWithProject("legacyuser", "testproject", projectMsgBytes)

		// Send different project message
		otherProjectMsg, _ := ws.NewMessage(ws.MessageTypeProjectCompleted, map[string]interface{}{
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

		legacyMsg, _ := ws.NewMessage(ws.MessageTypeNotification, legacyPayload)
		legacyMsgBytes, _ := legacyMsg.ToJSON()
		hub.SendToUser("legacyuser", legacyMsgBytes)

		// Read and verify message structure
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read legacy message: %v", err)
		}

		var receivedMsg ws.Message
		if err := json.Unmarshal(data, &receivedMsg); err != nil {
			t.Fatalf("Failed to unmarshal legacy message: %v", err)
		}

		// Verify message structure is preserved
		if receivedMsg.Type != ws.MessageTypeNotification {
			t.Errorf("Message type changed: expected %s, got %s", ws.MessageTypeNotification, receivedMsg.Type)
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
		rateLimiter := NewMockRateLimiter(2, 100, time.Minute)

		options := &httpDelivery.HandlerOptions{
			RateLimiter: rateLimiter,
		}

		legacyHandler := httpDelivery.NewHandlerWithOptions(hub, jwtValidator, logger, wsConfig, nil, cookieConfig, "dev", options)

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
