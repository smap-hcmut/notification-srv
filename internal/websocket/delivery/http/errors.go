package http

import (
	"net/http"

	"notification-srv/internal/websocket"
	"notification-srv/pkg/errors"
)

func (h *Handler) mapError(err error) error {
	switch err {
	case websocket.ErrInvalidToken:
		return errors.NewHTTPError(http.StatusUnauthorized, "Invalid or expired token")
	case websocket.ErrMissingToken:
		return errors.NewHTTPError(http.StatusUnauthorized, "Missing authentication token")
	case websocket.ErrMaxConnectionsReached:
		return errors.NewHTTPError(http.StatusServiceUnavailable, "Maximum connections reached")
	case websocket.ErrUserNotFound:
		return errors.NewHTTPError(http.StatusNotFound, "User not found")
	default:
		// Unknown errors panic to be caught by recovery middleware in development,
		// or logged as 500 in production.
		// Convention says: MUST panic on unknown errors.
		panic(err)
	}
}
