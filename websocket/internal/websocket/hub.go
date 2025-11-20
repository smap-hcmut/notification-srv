package websocket

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"smap-websocket/pkg/log"
)

// Hub maintains the set of active connections and broadcasts messages to them
type Hub struct {
	// Registered connections (userID -> []*Connection for multiple tabs)
	connections map[string][]*Connection
	mu          sync.RWMutex

	// Channels for connection management
	register   chan *Connection
	unregister chan *Connection

	// Channel for broadcasting messages
	broadcast chan *BroadcastMessage

	// Metrics
	totalConnections      atomic.Int64
	totalMessagesSent     atomic.Int64
	totalMessagesReceived atomic.Int64
	totalMessagesFailed   atomic.Int64

	// Configuration
	maxConnections int

	// Dependencies
	logger log.Logger

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewHub creates a new Hub instance
func NewHub(logger log.Logger, maxConnections int) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		connections:    make(map[string][]*Connection),
		register:       make(chan *Connection, 100),
		unregister:     make(chan *Connection, 100),
		broadcast:      make(chan *BroadcastMessage, 1000),
		maxConnections: maxConnections,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		done:           make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	defer close(h.done)

	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info(context.Background(), "Hub shutting down...")
			h.closeAllConnections()
			return

		case conn := <-h.register:
			h.registerConnection(conn)

		case conn := <-h.unregister:
			h.unregisterConnection(conn)

		case msg := <-h.broadcast:
			h.broadcastToUser(msg)
		}
	}
}

// registerConnection registers a new connection
func (h *Hub) registerConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check max connections limit
	if h.getTotalConnectionsLocked() >= h.maxConnections {
		h.logger.Warnf(context.Background(), "Max connections reached, rejecting user: %s", conn.userID)
		go conn.Close()
		return
	}

	// Add connection to user's connection list
	h.connections[conn.userID] = append(h.connections[conn.userID], conn)
	h.totalConnections.Add(1)

	h.logger.Infof(context.Background(),
		"User connected: %s (total connections: %d, user connections: %d)",
		conn.userID,
		h.getTotalConnectionsLocked(),
		len(h.connections[conn.userID]),
	)
}

// unregisterConnection unregisters a connection
func (h *Hub) unregisterConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	connections, exists := h.connections[conn.userID]
	if !exists {
		return
	}

	// Find and remove the connection
	for i, c := range connections {
		if c == conn {
			// Remove from slice
			h.connections[conn.userID] = append(connections[:i], connections[i+1:]...)
			h.totalConnections.Add(-1)

			// Close the connection
			close(conn.send)

			// If no more connections for this user, remove the user entry
			if len(h.connections[conn.userID]) == 0 {
				delete(h.connections, conn.userID)
				h.logger.Infof(context.Background(), "User disconnected (all tabs closed): %s", conn.userID)
			} else {
				h.logger.Infof(context.Background(),
					"User connection closed: %s (remaining connections: %d)",
					conn.userID,
					len(h.connections[conn.userID]),
				)
			}

			break
		}
	}
}

// broadcastToUser sends a message to all connections of a specific user
func (h *Hub) broadcastToUser(msg *BroadcastMessage) {
	h.mu.RLock()
	connections, exists := h.connections[msg.UserID]
	h.mu.RUnlock()

	if !exists || len(connections) == 0 {
		// User is not connected, skip silently (as per requirement H-09)
		return
	}

	// Convert message to JSON
	data, err := msg.Message.ToJSON()
	if err != nil {
		h.logger.Errorf(context.Background(), "Failed to marshal message: %v", err)
		h.totalMessagesFailed.Add(1)
		return
	}

	// Send to all user's connections
	sentCount := 0
	for _, conn := range connections {
		select {
		case conn.send <- data:
			sentCount++
		default:
			// Connection's send buffer is full, skip
			h.logger.Warnf(context.Background(), "Failed to send message to user %s (buffer full)", msg.UserID)
			h.totalMessagesFailed.Add(1)
		}
	}

	h.totalMessagesSent.Add(int64(sentCount))
	h.totalMessagesReceived.Add(1)
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID string, message *Message) {
	select {
	case h.broadcast <- &BroadcastMessage{
		UserID:  userID,
		Message: message,
	}:
	case <-time.After(time.Second):
		h.logger.Warnf(context.Background(), "Timeout sending message to user %s", userID)
		h.totalMessagesFailed.Add(1)
	}
}

// closeAllConnections closes all active connections
func (h *Hub) closeAllConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for userID, connections := range h.connections {
		for _, conn := range connections {
			conn.Close()
		}
		h.logger.Infof(context.Background(), "Closed all connections for user: %s", userID)
	}

	h.connections = make(map[string][]*Connection)
}

// GetStats returns hub statistics
func (h *Hub) GetStats() HubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return HubStats{
		ActiveConnections:     h.getTotalConnectionsLocked(),
		TotalUniqueUsers:      len(h.connections),
		TotalMessagesSent:     h.totalMessagesSent.Load(),
		TotalMessagesReceived: h.totalMessagesReceived.Load(),
		TotalMessagesFailed:   h.totalMessagesFailed.Load(),
	}
}

// getTotalConnectionsLocked returns total connections (must be called with lock held)
func (h *Hub) getTotalConnectionsLocked() int {
	total := 0
	for _, connections := range h.connections {
		total += len(connections)
	}
	return total
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown(ctx context.Context) error {
	h.cancel()

	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HubStats represents hub statistics
type HubStats struct {
	ActiveConnections     int   `json:"active_connections"`
	TotalUniqueUsers      int   `json:"total_unique_users"`
	TotalMessagesSent     int64 `json:"total_messages_sent"`
	TotalMessagesReceived int64 `json:"total_messages_received"`
	TotalMessagesFailed   int64 `json:"total_messages_failed"`
}
