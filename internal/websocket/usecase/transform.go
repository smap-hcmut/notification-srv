package usecase

import (
	"context"
	"encoding/json"
	"time"

	"notification-srv/internal/websocket"
)

// transformMessage transforms raw payload into a proper NotificationOutput based on message type.
func (uc *implUseCase) transformMessage(ctx context.Context, msgType websocket.MessageType, payload []byte) (websocket.NotificationOutput, error) {
	output := websocket.NotificationOutput{
		Type:      msgType,
		Timestamp: time.Now(),
	}

	switch msgType {
	case websocket.MessageTypeDataOnboarding:
		var data websocket.DataOnboardingPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return websocket.NotificationOutput{}, websocket.ErrInvalidMessage
		}
		// Validate/Transform logic if needed (e.g. strict status check)
		output.Payload = data

	case websocket.MessageTypeAnalyticsPipeline:
		var data websocket.AnalyticsPipelinePayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return websocket.NotificationOutput{}, websocket.ErrInvalidMessage
		}
		output.Payload = data

	case websocket.MessageTypeCrisisAlert:
		var data websocket.CrisisAlertPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return websocket.NotificationOutput{}, websocket.ErrInvalidMessage
		}
		output.Payload = data

	case websocket.MessageTypeCampaignEvent:
		var data websocket.CampaignEventPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return websocket.NotificationOutput{}, websocket.ErrInvalidMessage
		}
		output.Payload = data

	case websocket.MessageTypeSystem:
		// System messages might be plain strings or generic maps
		var data interface{}
		if err := json.Unmarshal(payload, &data); err != nil {
			return websocket.NotificationOutput{}, websocket.ErrInvalidMessage
		}
		output.Payload = data

	default:
		return websocket.NotificationOutput{}, websocket.ErrUnknownMessageType
	}

	return output, nil
}
