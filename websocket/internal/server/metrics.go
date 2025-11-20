package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	ws "smap-websocket/internal/websocket"
)

// MetricsResponse represents the metrics response
type MetricsResponse struct {
	Service     string            `json:"service"`
	Timestamp   time.Time         `json:"timestamp"`
	Uptime      int64             `json:"uptime_seconds"`
	Connections *ConnectionMetrics `json:"connections"`
	Messages    *MessageMetrics    `json:"messages"`
}

// ConnectionMetrics represents connection-related metrics
type ConnectionMetrics struct {
	Active           int `json:"active"`
	TotalUniqueUsers int `json:"total_unique_users"`
}

// MessageMetrics represents message-related metrics
type MessageMetrics struct {
	ReceivedFromRedis int64 `json:"received_from_redis"`
	SentToClients     int64 `json:"sent_to_clients"`
	Failed            int64 `json:"failed"`
}

// metricsHandler handles metrics requests
func metricsHandler(c *gin.Context, hub *ws.Hub) {
	stats := hub.GetStats()

	response := MetricsResponse{
		Service:   "websocket-service",
		Timestamp: time.Now(),
		Uptime:    int64(time.Since(startTime).Seconds()),
		Connections: &ConnectionMetrics{
			Active:           stats.ActiveConnections,
			TotalUniqueUsers: stats.TotalUniqueUsers,
		},
		Messages: &MessageMetrics{
			ReceivedFromRedis: stats.TotalMessagesReceived,
			SentToClients:     stats.TotalMessagesSent,
			Failed:            stats.TotalMessagesFailed,
		},
	}

	c.JSON(http.StatusOK, response)
}
