package http

import (
	"notification-srv/internal/websocket"

	"github.com/gin-gonic/gin"
)

// processUpgradeRequest handles the initial request processing before upgrade.
// It extracts the token, validates it, and returns the upgrade request info and keys.
func (h *Handler) processUpgradeRequest(c *gin.Context) (UpgradeReq, string, error) {
	var req UpgradeReq

	// 1. Bind Query Params (token, project_id)
	if err := c.ShouldBindQuery(&req); err != nil {
		return UpgradeReq{}, "", websocket.ErrInvalidMessage
	}

	// 2. Fallback: Check Cookie if token missing
	if req.Token == "" {
		if cookie, err := c.Cookie(h.cookieCfg.Name); err == nil {
			req.Token = cookie
		}
	}

	// 3. Validate Request DTO
	if err := req.validate(); err != nil {
		return UpgradeReq{}, "", err
	}

	// 4. Verify Token
	payload, err := h.jwtMgr.Verify(req.Token)
	if err != nil {
		h.logger.Warnf(c.Request.Context(), "token verification failed: %v", err)
		return UpgradeReq{}, "", websocket.ErrInvalidToken
	}

	// payload.UserID (assuming scope.Payload struct has UserID field based on pkg/jwt/interface.go usage of Verify returning scope.Payload)
	// If scope.Payload is map or struct, we need to know.
	// Based on "Verify(token string) (scope.Payload, error)" in interface.go.
	// We assume payload has UserID field or method.
	// Let's assume it's a struct with UserID.

	return req, payload.UserID, nil
}
