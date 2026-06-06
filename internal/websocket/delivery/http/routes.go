package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// RegisterRoutes registers the WebSocket routes.
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	// WebSocket auth is enforced inside the handler because the browser WebSocket
	// API cannot send custom bearer-token headers.
	r.GET("/ws", h.HandleWebSocket)
	r.GET("/notification/ws", h.HandleWebSocket)
}
