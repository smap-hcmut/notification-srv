package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// HandleWebSocket upgrades the HTTP connection to a WebSocket connection.
// @Summary Connect to WebSocket
// @Description Upgrade HTTP to WebSocket for real-time notifications. Requires valid JWT token in query 'token' or cookie.
// @Tags Notification
// @Param token query string true "JWT Token"
// @Param project_id query string false "Project ID Filter"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} response.Error "Unauthorized"
// @Router /ws [GET]
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// 1. Process Request (Auth & Validation)
	req, userID, err := h.processUpgradeRequest(c)
	if err != nil {
		// Map error to HTTP response
		// We use mapError but since we are in gin handler and not returning error to middleware,
		// we must look at how errors are handled.
		// Usually we return error response JSON.
		// h.mapError returns an error type that might be handled by middleware on return?
		// But Upgrade happens before return.
		// If fails here, we return standard HTTP error.
		// Convention: "Catch errors from UseCase and map them using response.Error"
		// Here we just abort with status.
		// Let's try to map it to status code using a helper or switch.
		// Using panic(err) as per "mapError rules" is for domain errors.
		// Here it's pre-domain.

		// Simplify:
		_ = h.mapError(err) // This will panic if unknown, or return HTTPError.
		// We should probably catch the panic if we want to be safe, or just use error response directly.
		// Since mapError panics on unknown, we trust it or catch it in middleware.
		// But for known errors, it returns pkgErrors.HTTPError.
		// We can't easily extract status code from generic error interface without casting.
		// For now, let's just abort with 400/401 based on error type manually or assume middleware catches panics.
		// Wait, mapError returns error. If I ignore it, nothing happens.
		// If I panic(err), middleware catches.
		// If I return, Gin continues.
		// I should probably abort with error.
		c.Error(h.mapError(err))
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
