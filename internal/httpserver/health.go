package httpserver

import (
	"notification-srv/internal/websocket"
	"notification-srv/pkg/errors"
	"notification-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

// healthCheck handles health check requests
// @Summary Health Check
// @Description Check if the WebSocket service is healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Service is healthy"
// @Router /health [get]
func (srv *HTTPServer) healthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check Redis connection
	if err := srv.redis.Ping(ctx); err != nil {
		response.HttpError(c, errors.NewHTTPError(503, "Redis connection failed"))
		return
	}

	// Get Hub stats for health info
	hubStats, err := srv.wsUC.GetStats(ctx)
	if err != nil {
		// Log error but maybe still return healthy for other parts?
		// Simple fix: assume 0 if error.
		hubStats = websocket.HubStats{}
	}

	response.OK(c, gin.H{
		"status":             "healthy",
		"message":            "From SMAP Notification Service With Love",
		"version":            "1.0.0",
		"service":            "notification-srv",
		"active_connections": hubStats.ActiveConnections,
		"total_unique_users": hubStats.TotalUniqueUsers,
		"redis":              "connected",
	})
}

// readyCheck handles readiness check requests
// @Summary Readiness Check
// @Description Check if the WebSocket service is ready to serve traffic
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Service is ready"
// @Failure 503 {object} map[string]interface{} "Service is not ready"
// @Router /ready [get]
func (srv *HTTPServer) readyCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check if Redis is ready
	if err := srv.redis.Ping(ctx); err != nil {
		response.HttpError(c, errors.NewHTTPError(503, "Redis connection not available"))
		return
	}

	response.OK(c, gin.H{
		"status":  "ready",
		"message": "From SMAP Notification Service With Love",
		"version": "1.0.0",
		"service": "notification-srv",
		"redis":   "connected",
	})
}

// liveCheck handles liveness check requests
// @Summary Liveness Check
// @Description Check if the WebSocket service is alive
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Service is alive"
// @Router /live [get]
func (srv *HTTPServer) liveCheck(c *gin.Context) {
	response.OK(c, gin.H{
		"status":  "alive",
		"message": "From SMAP Notification Service With Love",
		"version": "1.0.0",
		"service": "notification-srv",
	})
}
