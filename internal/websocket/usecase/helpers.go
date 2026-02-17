package usecase

import (
	"encoding/json"
	"strings"

	"notification-srv/internal/websocket"
)

// parseChannel parses a Redis channel string into a ParsedChannel struct.
// Supported formats:
// - project:{project_id}:user:{user_id}
// - campaign:{campaign_id}:user:{user_id}
// - alert:{subtype}:user:{user_id}
// - system:{subtype}
func parseChannel(channel string) (ParsedChannel, error) {
	parts := strings.Split(channel, ":")
	if len(parts) < 2 {
		return ParsedChannel{}, websocket.ErrInvalidChannel
	}

	result := ParsedChannel{}

	switch parts[0] {
	case "project":
		if len(parts) != 4 || parts[2] != "user" {
			return ParsedChannel{}, websocket.ErrInvalidChannel
		}
		result.ChannelType = websocket.ChannelTypeProject
		result.EntityID = parts[1]
		result.UserID = parts[3]

	case "campaign":
		if len(parts) != 4 || parts[2] != "user" {
			return ParsedChannel{}, websocket.ErrInvalidChannel
		}
		result.ChannelType = websocket.ChannelTypeCampaign
		result.EntityID = parts[1]
		result.UserID = parts[3]

	case "alert":
		// alert:crisis:user:{user_id}
		if len(parts) != 4 || parts[2] != "user" {
			return ParsedChannel{}, websocket.ErrInvalidChannel
		}
		result.ChannelType = websocket.ChannelTypeAlert
		result.SubType = parts[1]
		result.UserID = parts[3]

	case "system":
		// system:maintenance
		result.ChannelType = websocket.ChannelTypeSystem
		result.SubType = parts[1]

	default:
		return ParsedChannel{}, websocket.ErrInvalidChannel
	}

	return result, nil
}

// detectMessageType unmarshals the payload partially to detect or infer the message type.
// For now, we assume the message type is inferred from the structure or passed in metadata.
// However, based on the proposal, we can try to unmarshal to known types or check fields.
// Simplified approach: try to unmarshal to a generic map and check unique fields or tags?
// Better approach: Rely on the structure or distinct fields.
// For this strict implementation, let's assume specific unique fields.
func detectMessageType(payload []byte) (websocket.MessageType, error) {
	var partial map[string]interface{}
	if err := json.Unmarshal(payload, &partial); err != nil {
		return "", err
	}

	// Heuristics based on unique fields
	if _, ok := partial["source_id"]; ok {
		// Could be DataOnboarding or AnalyticsPipeline
		// Check for specific fields
		if _, ok := partial["total_records"]; ok {
			return websocket.MessageTypeAnalyticsPipeline, nil
		}
		if _, ok := partial["record_count"]; ok {
			return websocket.MessageTypeDataOnboarding, nil
		}
	}

	if _, ok := partial["alert_type"]; ok {
		return websocket.MessageTypeCrisisAlert, nil
	}

	if _, ok := partial["campaign_id"]; ok {
		return websocket.MessageTypeCampaignEvent, nil
	}

	if _, ok := partial["system_event"]; ok {
		return websocket.MessageTypeSystem, nil
	}

	return "", websocket.ErrUnknownMessageType
}
