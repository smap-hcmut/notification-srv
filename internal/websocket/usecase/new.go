package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"

	"notification-srv/internal/alert"
	ws "notification-srv/internal/websocket"
	"notification-srv/pkg/log"
)

// implUseCase implements websocket.UseCase.
type implUseCase struct {
	hub            *Hub
	logger         log.Logger
	alertUC        alert.UseCase
	maxConnections int
}

// New creates a new WebSocket UseCase.
func New(logger log.Logger, maxConnections int, alertUC alert.UseCase) ws.UseCase {
	hub := newHub(logger, maxConnections)
	return &implUseCase{
		hub:            hub,
		logger:         logger,
		alertUC:        alertUC,
		maxConnections: maxConnections,
	}
}

func (uc *implUseCase) Run() {
	uc.hub.run()
}

func (uc *implUseCase) Shutdown(ctx context.Context) error {
	// Implement graceful shutdown of hub if needed
	return nil
}

func (uc *implUseCase) Register(ctx context.Context, input ws.ConnectionInput) error {
	conn, ok := input.Conn.(*websocket.Conn)
	if !ok {
		return fmt.Errorf("invalid connection type")
	}

	client := &Connection{
		hub:    uc.hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: input.UserID,
	}

	uc.hub.register <- client

	// Start the pumps
	go client.writePump(uc.logger)
	go client.readPump()

	return nil
}

func (uc *implUseCase) Unregister(ctx context.Context, input ws.ConnectionInput) error {
	// Not typically called manually from outside, handled by readPump closure
	return nil
}

func (uc *implUseCase) GetStats(ctx context.Context) (ws.HubStats, error) {
	active, unique := uc.hub.Stats()
	return ws.HubStats{
		ActiveConnections: active,
		TotalUniqueUsers:  unique,
	}, nil
}

func (uc *implUseCase) ProcessMessage(ctx context.Context, input ws.ProcessMessageInput) error {
	// 1. Parse channel
	parsed, err := parseChannel(input.Channel)
	if err != nil {
		uc.logger.Warnf(ctx, "parse channel failed: %v", err)
		return nil // Swallow error to avoid spamming logs/retries for invalid channels
	}

	// 2. Detect message type
	msgType, err := detectMessageType(input.Payload)
	if err != nil {
		uc.logger.Warnf(ctx, "detect type failed: %v", err) // Log info/warn
		// We might fail here or default to SYSTEM? For now return error
		return nil
	}

	// 3. Validate & Transform
	output, err := uc.transformMessage(ctx, msgType, input.Payload)
	if err != nil {
		return fmt.Errorf("transform: %w", err)
	}

	// 4. Dispatch to alert channel (Discord) if needed
	// Note: We use the alertUC for this.
	// Logic: If it is a crisis alert, dispatch it.
	if msgType == ws.MessageTypeCrisisAlert {
		// Needs unmarshaling payload to CrisisAlertPayload to pass to DispatchCrisisAlert
		// transformMessage already did that but returned NotificationOutput.Payload as interface{}
		if payloadData, ok := output.Payload.(ws.CrisisAlertPayload); ok {
			// Map to alert.CrisisAlertInput
			alertInput := alert.CrisisAlertInput{
				ProjectID:       payloadData.ProjectID,
				ProjectName:     payloadData.ProjectName,
				Severity:        payloadData.Severity,
				AlertType:       payloadData.AlertType,
				Metric:          payloadData.Metric,
				CurrentValue:    payloadData.CurrentValue,
				Threshold:       payloadData.Threshold,
				AffectedAspects: payloadData.AffectedAspects,
				SampleMentions:  payloadData.SampleMentions,
				TimeWindow:      payloadData.TimeWindow,
				ActionRequired:  payloadData.ActionRequired,
				GeneratedAt:     output.Timestamp,
			}

			go func() {
				if err := uc.alertUC.DispatchCrisisAlert(context.Background(), alertInput); err != nil {
					uc.logger.Warnf(ctx, "alert dispatch failed: %v", err)
				}
			}()
		}
	}

	// 5. Route to WebSocket connections
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}

	uc.routeMessage(parsed, outputBytes)
	return nil
}

func (uc *implUseCase) routeMessage(parsed ParsedChannel, message []byte) {
	// Broad strategy:
	// If UserID is present, send to that user.
	// If UserID is empty, it might be a broadcast (e.g. system wide).
	// Currently our parsing logic enforces UserID for most types except System.

	if parsed.UserID != "" {
		uc.hub.SendToUser(parsed.UserID, message)
	} else if parsed.ChannelType == ws.ChannelTypeSystem {
		uc.hub.Broadcast(message)
	}
}

func (uc *implUseCase) OnUserConnected(ctx context.Context, userID string) error {
	// Implementation hook for metrics or other side effects
	return nil
}

func (uc *implUseCase) OnUserDisconnected(ctx context.Context, userID string, hasOtherConnections bool) error {
	// Implementation hook
	return nil
}
