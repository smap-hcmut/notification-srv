package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/smap-hcmut/shared-libs/go/response"
)

// HandleWebSocket upgrades the HTTP connection to a WebSocket connection.
// @Summary Connect to WebSocket
// @Description Upgrade HTTP to WebSocket for real-time notifications. Requires valid JWT token in query 'token' or cookie.
// @Tags Notification
// @Param token query string true "JWT Token"
// @Param project_id query string false "Project ID Filter"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} response.Resp "Unauthorized"
// @Router /ws [GET]
func (h *handler) HandleWebSocket(c *gin.Context) {
	// 1. Process Request (Auth & Validation)
	req, userID, err := h.processUpgradeRequest(c)
	if err != nil {
		// Map domain error to HTTP error and send response
		response.Error(c, h.mapError(err))
		return
	}

	// 2. Upgrade Connection
	upgrader := websocket.Upgrader{
		ReadBufferSize:  h.wsConfig.ReadBufferSize,
		WriteBufferSize: h.wsConfig.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			// Check against allowed origins or return true for now
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf(c.Request.Context(), "upgrade failed: %v", err)
		return
	}

	// 3. Register Connection via UseCase
	input := req.toInput(conn, userID)
	if err := h.uc.Register(c.Request.Context(), input); err != nil {
		h.logger.Errorf(c.Request.Context(), "register failed: %v", err)
		conn.Close()
		return
	}

	// Connection is now managed by UseCase (Hub).
	// We don't need to do anything else here.
}
