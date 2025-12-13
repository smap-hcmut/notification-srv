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

	// Topic subscription filters (empty string means no filter/all messages)
	projectID string // Filter for project-specific messages
	jobID     string // Filter for job-specific messages

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

// ConnectionOptions holds optional parameters for creating a connection
type ConnectionOptions struct {
	ProjectID string // Optional project filter
	JobID     string // Optional job filter
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
		projectID:  "",
		jobID:      "",
		send:       make(chan []byte, 256),
		pongWait:   pongWait,
		pingPeriod: pingPeriod,
		writeWait:  writeWait,
		logger:     logger,
		done:       make(chan struct{}),
	}
}

// NewConnectionWithFilters creates a new Connection instance with topic filters
func NewConnectionWithFilters(
	hub *Hub,
	conn *websocket.Conn,
	userID string,
	projectID string,
	jobID string,
	pongWait time.Duration,
	pingPeriod time.Duration,
	writeWait time.Duration,
	logger log.Logger,
) *Connection {
	return &Connection{
		hub:        hub,
		conn:       conn,
		userID:     userID,
		projectID:  projectID,
		jobID:      jobID,
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
		if c.conn != nil {
			c.conn.Close()
		}
	}
}

// GetProjectID returns the project ID filter for this connection
func (c *Connection) GetProjectID() string {
	return c.projectID
}

// GetJobID returns the job ID filter for this connection
func (c *Connection) GetJobID() string {
	return c.jobID
}

// GetUserID returns the user ID for this connection
func (c *Connection) GetUserID() string {
	return c.userID
}

// HasProjectFilter returns true if this connection has a project filter
func (c *Connection) HasProjectFilter() bool {
	return c.projectID != ""
}

// HasJobFilter returns true if this connection has a job filter
func (c *Connection) HasJobFilter() bool {
	return c.jobID != ""
}

// HasTopicFilter returns true if this connection has any topic filter
func (c *Connection) HasTopicFilter() bool {
	return c.projectID != "" || c.jobID != ""
}

// MatchesProject returns true if this connection should receive messages for the given project
// Returns true if no project filter is set (receives all) or if the project matches
func (c *Connection) MatchesProject(projectID string) bool {
	return c.projectID == "" || c.projectID == projectID
}

// MatchesJob returns true if this connection should receive messages for the given job
// Returns true if no job filter is set (receives all) or if the job matches
func (c *Connection) MatchesJob(jobID string) bool {
	return c.jobID == "" || c.jobID == jobID
}
