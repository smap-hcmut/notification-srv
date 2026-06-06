package httpserver

import (
	"notification-srv/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
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

	redisStatus := "disconnected"
	if srv.redis != nil {
		if err := srv.redis.Ping(ctx); err == nil {
			redisStatus = "connected"
		}
	}

	// Check Redis connection
	if srv.redis == nil {
		// Not fatal for liveness; service can still run in degraded mode.
		redisStatus = "disconnected"
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
		"redis":              redisStatus,
	})
}

// readyCheck handles readiness check requests.
//
// Redis drives realtime fanout, but the service can still accept HTTP and
// WebSocket connections in degraded mode. Returning 200 with an explicit
// degraded status keeps Kubernetes from hiding the service while preserving
// operator-visible Redis failure details.
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

	redisStatus := "connected"
	redisErr := ""
	if srv.redis == nil {
		redisStatus = "degraded"
		redisErr = "not initialized"
	} else if err := srv.redis.Ping(ctx); err != nil {
		redisStatus = "degraded"
		redisErr = err.Error()
	}

	subscriberStatus := "ready"
	if srv.wsSubscriber == nil {
		subscriberStatus = "disabled"
	}

	status := "ready"
	if redisStatus != "connected" || subscriberStatus != "ready" {
		status = "degraded"
	}

	response.OK(c, gin.H{
		"status":      status,
		"message":     "From SMAP Notification Service With Love",
		"version":     "1.0.0",
		"service":     "notification-srv",
		"redis":       redisStatus,
		"redis_error": redisErr,
		"subscriber":  subscriberStatus,
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
