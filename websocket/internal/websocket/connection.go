package websocket

import (
	"context"
	"time"

	"smap-websocket/pkg/log"

	"github.com/gorilla/websocket"
)

// Connection represents a WebSocket connection for a user
type Connection struct {
	// Hub reference
	hub *Hub

	// WebSocket connection
	conn *websocket.Conn

	// User ID from JWT
	userID string

	// Buffered channel of outbound messages
	send chan []byte

	// Configuration
	pongWait   time.Duration
	pingPeriod time.Duration
	writeWait  time.Duration

	// Logger
	logger log.Logger

	// Done signal
	done chan struct{}
}

// NewConnection creates a new Connection instance
func NewConnection(
	hub *Hub,
	conn *websocket.Conn,
	userID string,
	pongWait time.Duration,
	pingPeriod time.Duration,
	writeWait time.Duration,
	logger log.Logger,
) *Connection {
	return &Connection{
		hub:        hub,
		conn:       conn,
		userID:     userID,
		send:       make(chan []byte, 256),
		pongWait:   pongWait,
		pingPeriod: pingPeriod,
		writeWait:  writeWait,
		logger:     logger,
		done:       make(chan struct{}),
	}
}

// readPump pumps messages from the WebSocket connection to the hub
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Connection) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline for pong messages
	c.conn.SetReadDeadline(time.Now().Add(c.pongWait))

	// Set pong handler
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
		return nil
	})

	// Set max message size
	c.conn.SetReadLimit(512)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Errorf(context.Background(), "WebSocket read error for user %s: %v", c.userID, err)
			}
			break
		}

		// Log received message (optional, for debugging)
		c.logger.Debugf(context.Background(), "Received message from user %s: %s", c.userID, string(message))

		// Note: In this service, we don't process incoming messages from clients
		// as per the requirement (H-09: only push messages to clients)
		// But we keep the read pump running to detect disconnections and handle pong messages
	}
}

// writePump pumps messages from the hub to the WebSocket connection
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Connection) writePump() {
	ticker := time.NewTicker(c.pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			// Set write deadline
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))

			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write message
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping message
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// Start starts the connection's read and write pumps
func (c *Connection) Start() {
	go c.writePump()
	go c.readPump()
}

// Close closes the connection
func (c *Connection) Close() {
	select {
	case <-c.done:
		// Already closed
		return
	default:
		close(c.done)
		c.conn.Close()
	}
}
