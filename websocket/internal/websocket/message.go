package websocket

import (
	"encoding/json"
	"time"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeNotification     MessageType = "notification"
	MessageTypeAlert            MessageType = "alert"
	MessageTypeUpdate           MessageType = "update"
	MessageTypePing             MessageType = "ping"
	MessageTypePong             MessageType = "pong"
	MessageTypeProjectProgress  MessageType = "project_progress"
	MessageTypeProjectCompleted MessageType = "project_completed"
	MessageTypeDryRunResult     MessageType = "dryrun_result"
)

// ProgressPayload represents project progress notification from Project Service
type ProgressPayload struct {
	ProjectID       string  `json:"project_id"`
	Status          string  `json:"status"`
	Total           int     `json:"total"`
	Done            int     `json:"done"`
	Errors          int     `json:"errors"`
	ProgressPercent float64 `json:"progress_percent"`
}

// Validate validates the progress payload fields
func (p *ProgressPayload) Validate() error {
	if p.ProjectID == "" {
		return ErrInvalidMessage
	}
	if p.Total < 0 || p.Done < 0 || p.Errors < 0 {
		return ErrInvalidMessage
	}
	if p.ProgressPercent < 0 || p.ProgressPercent > 100 {
		return ErrInvalidMessage
	}
	return nil
}

// Message represents a message sent over WebSocket
type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// BroadcastMessage represents a message to be broadcast to specific users
type BroadcastMessage struct {
	UserID  string
	Message *Message
}

// NewMessage creates a new message with the given type and payload
func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
	}, nil
}

// ToJSON converts the message to JSON bytes
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON creates a message from JSON bytes
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// Validate validates the message structure
func (m *Message) Validate() error {
	if m.Type == "" {
		return ErrInvalidMessage
	}
	if m.Payload == nil {
		return ErrInvalidMessage
	}
	return nil
}
