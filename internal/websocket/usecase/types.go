package usecase

import (
	"notification-srv/internal/websocket"
)

// ParsedChannel represents the components extracted from a Redis channel string.
type ParsedChannel struct {
	ChannelType websocket.ChannelType
	EntityID    string // project_id, campaign_id, etc.
	UserID      string // Target user (empty for broadcast channels like system:*)
	SubType     string // For alert channels: "crisis", "warning"
}
