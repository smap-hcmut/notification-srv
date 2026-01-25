package websocket

import "errors"

var (
	// ErrInvalidToken is returned when the JWT token is invalid
	ErrInvalidToken = errors.New("invalid or expired token")

	// ErrMissingToken is returned when the JWT token is missing
	ErrMissingToken = errors.New("missing token")

	// ErrInvalidMessage is returned when the message format is invalid
	ErrInvalidMessage = errors.New("invalid message format")

	// ErrConnectionClosed is returned when trying to write to a closed connection
	ErrConnectionClosed = errors.New("connection closed")

	// ErrMaxConnectionsReached is returned when max connections limit is reached
	ErrMaxConnectionsReached = errors.New("maximum connections reached")

	// ErrUserNotFound is returned when user is not found in registry
	ErrUserNotFound = errors.New("user not found in registry")
)
