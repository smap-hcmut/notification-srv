package websocket

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"smap-websocket/pkg/jwt"
	"smap-websocket/pkg/log"
)

// Production origins allowed in all environments
var productionOrigins = []string{
	"https://smap.tantai.dev",
	"https://smap-api.tantai.dev",
	"http://smap.tantai.dev",     // For testing/non-HTTPS
	"http://smap-api.tantai.dev", // For testing/non-HTTPS
}

// Private subnets allowed in dev/staging environments
var privateSubnets = []string{
	"172.16.21.0/24", // K8s cluster subnet
	"172.16.19.0/24", // Private network 1
	"192.168.1.0/24", // Private network 2
}

// isPrivateOrigin checks if an origin URL's hostname is in a configured private subnet
func isPrivateOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	hostname := u.Hostname()
	// Remove port if present
	if strings.Contains(hostname, ":") {
		hostname = strings.Split(hostname, ":")[0]
	}

	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}

	for _, cidr := range privateSubnets {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}

// isLocalhostOrigin checks if an origin is localhost or 127.0.0.1
func isLocalhostOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	hostname := u.Hostname()
	return hostname == "localhost" || hostname == "127.0.0.1"
}

// createUpgrader creates a WebSocket upgrader with environment-aware CORS validation
func createUpgrader(environment string) websocket.Upgrader {
	// Default to production if environment is empty
	if environment == "" {
		environment = "production"
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	if environment == "production" {
		// Production mode: static list only
		upgrader.CheckOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			for _, allowed := range productionOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		}
	} else {
		// Dev/Staging mode: dynamic validation
		upgrader.CheckOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")

			// Check production domains
			for _, allowed := range productionOrigins {
				if origin == allowed {
					return true
				}
			}

			// Check localhost
			if isLocalhostOrigin(origin) {
				return true
			}

			// Check private subnets
			if isPrivateOrigin(origin) {
				return true
			}

			return false
		}
	}

	return upgrader
}

// Handler handles WebSocket connections
type Handler struct {
	hub           *Hub
	jwtValidator  *jwt.Validator
	logger        log.Logger
	wsConfig      WSConfig
	redisNotifier RedisNotifier
	cookieConfig  CookieConfig
	environment   string
	upgrader      websocket.Upgrader
}

// WSConfig holds WebSocket configuration
type WSConfig struct {
	PongWait       time.Duration
	PingPeriod     time.Duration
	WriteWait      time.Duration
	MaxMessageSize int64
}

// CookieConfig holds cookie authentication configuration
type CookieConfig struct {
	Domain         string
	Secure         bool
	SameSite       string
	MaxAge         int
	MaxAgeRemember int
	Name           string
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
	cookieConfig CookieConfig,
	environment string,
) *Handler {
	// Log CORS mode on startup
	ctx := context.Background()
	if environment == "" {
		environment = "production"
	}
	if environment == "production" {
		logger.Infof(ctx, "CORS mode: production (strict origins only)")
	} else {
		logger.Infof(ctx, "CORS mode: %s (permissive - allows localhost and private subnets)", environment)
	}

	return &Handler{
		hub:           hub,
		jwtValidator:  jwtValidator,
		logger:        logger,
		wsConfig:      wsConfig,
		redisNotifier: redisNotifier,
		cookieConfig:  cookieConfig,
		environment:   environment,
		upgrader:      createUpgrader(environment),
	}
}

// HandleWebSocket handles WebSocket connection requests
// Implements requirements H-01, H-02, H-03, H-04, H-05
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// H-02: Extract JWT from cookie (primary method) or query parameter (fallback)
	// Priority: Cookie first, then query parameter for backward compatibility
	token, err := c.Cookie(h.cookieConfig.Name)
	if err != nil || token == "" {
		// Fallback to query parameter for backward compatibility
		token = c.Query("token")
		if token != "" {
			h.logger.Warn(context.Background(), "WebSocket connection using deprecated query parameter authentication")
		}
	}

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
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
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
