package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"smap-websocket/pkg/jwt"
	"smap-websocket/pkg/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for now (configure in production)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler handles WebSocket connections
type Handler struct {
	hub           *Hub
	jwtValidator  *jwt.Validator
	logger        log.Logger
	wsConfig      WSConfig
	redisNotifier RedisNotifier
}

// WSConfig holds WebSocket configuration
type WSConfig struct {
	PongWait       time.Duration
	PingPeriod     time.Duration
	WriteWait      time.Duration
	MaxMessageSize int64
}

// RedisNotifier is an interface for notifying Redis about connection changes
type RedisNotifier interface {
	OnUserConnected(userID string) error
	OnUserDisconnected(userID string, hasOtherConnections bool) error
}

// NewHandler creates a new WebSocket handler
func NewHandler(
	hub *Hub,
	jwtValidator *jwt.Validator,
	logger log.Logger,
	wsConfig WSConfig,
	redisNotifier RedisNotifier,
) *Handler {
	return &Handler{
		hub:           hub,
		jwtValidator:  jwtValidator,
		logger:        logger,
		wsConfig:      wsConfig,
		redisNotifier: redisNotifier,
	}
}

// HandleWebSocket handles WebSocket connection requests
// Implements requirements H-01, H-02, H-03, H-04, H-05
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// H-02: Extract JWT from query parameter
	token := c.Query("token")
	if token == "" {
		h.logger.Warn(context.Background(), "WebSocket connection rejected: missing token")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing token parameter",
		})
		return
	}

	// H-03: Validate JWT and extract user ID
	userID, err := h.jwtValidator.ExtractUserID(token)
	if err != nil {
		// H-04: Reject with 401 if token is invalid
		h.logger.Warnf(context.Background(), "WebSocket connection rejected: invalid token - %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired token",
		})
		return
	}

	// H-01: Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf(context.Background(), "Failed to upgrade connection: %v", err)
		return
	}

	// H-05: Create and register connection in Hub
	connection := NewConnection(
		h.hub,
		conn,
		userID,
		h.wsConfig.PongWait,
		h.wsConfig.PingPeriod,
		h.wsConfig.WriteWait,
		h.logger,
	)

	// Register connection with hub
	h.hub.register <- connection

	// Notify Redis subscriber that user is connected
	if h.redisNotifier != nil {
		if err := h.redisNotifier.OnUserConnected(userID); err != nil {
			h.logger.Errorf(context.Background(), "Failed to notify Redis about connection: %v", err)
		}
	}

	// Start connection pumps (read and write)
	connection.Start()

	h.logger.Infof(context.Background(), "WebSocket connection established for user: %s", userID)
}

// SetupRoutes sets up WebSocket routes
func (h *Handler) SetupRoutes(router *gin.Engine) {
	router.GET("/ws", h.HandleWebSocket)
}
