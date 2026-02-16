package discord

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"smap-websocket/pkg/log"
)

// DiscordWebhook contains webhook information for Discord API.
type DiscordWebhook struct {
	ID    string
	Token string
}

// NewDiscordWebhook creates a new Discord webhook instance.
func NewDiscordWebhook(id, token string) (*DiscordWebhook, error) {
	if id == "" || token == "" {
		return nil, errors.New("id and token are required")
	}

	return &DiscordWebhook{
		ID:    id,
		Token: token,
	}, nil
}

// Discord is the Discord service implementation for sending webhook messages.
type Discord struct {
	l       log.Logger
	webhook *DiscordWebhook
	config  Config
	client  *http.Client
}

// New creates a new Discord service instance with the provided logger and webhook.
func New(l log.Logger, webhook *DiscordWebhook) (*Discord, error) {
	if webhook == nil {
		return nil, errors.New("webhook is required")
	}

	if webhook.ID == "" || webhook.Token == "" {
		return nil, errors.New("webhook ID and token are required")
	}

	config := DefaultConfig()

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	return &Discord{
		l:       l,
		webhook: webhook,
		config:  config,
		client:  client,
	}, nil
}

// GetWebhookURL returns the Discord webhook URL.
func (d *Discord) GetWebhookURL() string {
	return fmt.Sprintf(webhookURL, d.webhook.ID, d.webhook.Token)
}

// Close closes idle connections in the HTTP client.
func (d *Discord) Close() error {
	d.client.CloseIdleConnections()
	return nil
}

// DiscordService interface defines methods for Discord service.
type DiscordService interface {
	SendMessage(ctx context.Context, content string) error
	SendEmbed(ctx context.Context, options MessageOptions) error
	SendError(ctx context.Context, title, description string, err error) error
	SendSuccess(ctx context.Context, title, description string) error
	SendWarning(ctx context.Context, title, description string) error
	SendInfo(ctx context.Context, title, description string) error
	ReportBug(ctx context.Context, message string) error
}
