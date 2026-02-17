package http

import (
	"notification-srv/internal/websocket"

	"github.com/gorilla/websocket"
)

// --- Configuration DTOs ---

type WSConfig struct {
	MaxConnections  int
	ReadBufferSize  int
	WriteBufferSize int
	AllowedOrigins  []string
}

type CookieConfig struct {
	Name     string
	Domain   string
	Path     string
	Secure   bool
	HttpOnly bool
	MaxAge   int
}

// --- Request DTOs ---

type UpgradeReq struct {
	Token     string `form:"token"`
	ProjectID string `form:"project_id"`
}

func (r UpgradeReq) validate() error {
	if r.Token == "" {
		return websocket.ErrMissingToken
	}
	// ProjectID is optional filter
	return nil
}

// toInput maps the DTO and connection to the UseCase input.
// Note: We cast *websocket.Conn to interface{} here.
func (r UpgradeReq) toInput(conn *websocket.Conn, userID string) websocket.ConnectionInput {
	return websocket.ConnectionInput{
		UserID:    userID,
		ProjectID: r.ProjectID,
		Conn:      conn,
	}
}
