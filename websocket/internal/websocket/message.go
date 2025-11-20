package websocket

import (
	"encoding/json"
	"time"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeNotification MessageType = "notification"
	MessageTypeAlert        MessageType = "alert"
	MessageTypeUpdate       MessageType = "update"
	MessageTypePing         MessageType = "ping"
	MessageTypePong         MessageType = "pong"
)

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
