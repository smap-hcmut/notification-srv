package websocket

import (
	"context"
)

// UseCase defines the business logic for the WebSocket domain.
// It combines connection management and message processing/transformation.
type UseCase interface {
	// Lifecycle
	Run()
	Shutdown(ctx context.Context) error

	// Connection Management
	// Note: Register takes a Connection interface/struct defined in types.go or internal
	Register(ctx context.Context, input ConnectionInput) error
	Unregister(ctx context.Context, input ConnectionInput) error

	// Stats
	GetStats(ctx context.Context) (HubStats, error)

	// Message Processing (Call by Redis Delivery or HTTP)
	// Validates, Transforms, and Routes message to connected users
	ProcessMessage(ctx context.Context, input ProcessMessageInput) error

	// Event Callbacks (Call by Redis Delivery)
	OnUserConnected(ctx context.Context, userID string) error
	OnUserDisconnected(ctx context.Context, userID string, hasOtherConnections bool) error
}
