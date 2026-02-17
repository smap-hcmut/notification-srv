package websocket

import "errors"

var (
	ErrInvalidToken          = errors.New("invalid or expired JWT token")
	ErrMissingToken          = errors.New("missing JWT token")
	ErrConnectionClosed      = errors.New("connection closed")
	ErrMaxConnectionsReached = errors.New("maximum connections reached")
	ErrUserNotFound          = errors.New("user not found in connection registry")
)

// Message errors
var (
	ErrInvalidMessage     = errors.New("invalid message format")
	ErrUnknownMessageType = errors.New("unknown message type")
	ErrInvalidChannel     = errors.New("invalid Redis channel format")
)

// Transform errors
var (
	ErrTransformFailed  = errors.New("message transformation failed")
	ErrValidationFailed = errors.New("message validation failed")
)
