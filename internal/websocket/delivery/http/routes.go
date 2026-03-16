package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// RegisterRoutes registers the WebSocket routes.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	// WebSocket endpoint
	// Note: We might allow public access to /ws but enforce auth inside handler,
	// because browser's WebSocket API doesn't allow custom headers for bearer token.
	// So we might skip standard auth middleware here if it strictly requires Header.

	ws := r.Group("/ws")
	{
		ws.GET("", h.HandleWebSocket)
	}
}
