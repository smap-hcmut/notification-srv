package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	ws "smap-websocket/internal/websocket"
	"smap-websocket/pkg/log"
	"smap-websocket/pkg/redis"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string         `json:"status"`
	Timestamp time.Time      `json:"timestamp"`
	Redis     *RedisHealth   `json:"redis"`
	WebSocket *WebSocketInfo `json:"websocket"`
	Uptime    int64          `json:"uptime_seconds"`
}

// RedisHealth represents Redis health status
type RedisHealth struct {
	Status  string  `json:"status"`
	PingMs  float64 `json:"ping_ms,omitempty"`
	Error   string  `json:"error,omitempty"`
}

// WebSocketInfo represents WebSocket server info
type WebSocketInfo struct {
	ActiveConnections int `json:"active_connections"`
	TotalUniqueUsers  int `json:"total_unique_users"`
}

var startTime = time.Now()

// healthHandler handles health check requests
func healthHandler(c *gin.Context, logger log.Logger, hub *ws.Hub, redisClient *redis.Client) {
	ctx := context.Background()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    int64(time.Since(startTime).Seconds()),
	}

	// Check Redis health
	redisHealth := &RedisHealth{
		Status: "connected",
	}

	pingDuration, err := redisClient.Ping(ctx)
	if err != nil {
		redisHealth.Status = "disconnected"
		redisHealth.Error = err.Error()
		response.Status = "degraded"
		logger.Errorf(ctx, "Redis health check failed: %v", err)
	} else {
		redisHealth.PingMs = float64(pingDuration.Microseconds()) / 1000.0
	}

	response.Redis = redisHealth

	// Get WebSocket stats
	stats := hub.GetStats()
	response.WebSocket = &WebSocketInfo{
		ActiveConnections: stats.ActiveConnections,
		TotalUniqueUsers:  stats.TotalUniqueUsers,
	}

	// Return appropriate status code
	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}
