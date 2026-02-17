package discord

import (
	"net/http"
	"time"

	"notification-srv/pkg/log"
)

type Config struct {
	Timeout          time.Duration
	RetryCount       int
	RetryDelay       time.Duration
	DefaultUsername  string
	DefaultAvatarURL string
}

type webhookInfo struct {
	id    string
	token string
}

type discordImpl struct {
	l       log.Logger
	webhook *webhookInfo
	config  Config
	client  *http.Client
}

type MessageType string

const (
	MessageTypeInfo    MessageType = "info"
	MessageTypeSuccess MessageType = "success"
	MessageTypeWarning MessageType = "warning"
	MessageTypeError   MessageType = "error"
)

type MessageLevel int

const (
	LevelLow MessageLevel = iota
	LevelNormal
	LevelHigh
	LevelUrgent
)

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type EmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type Embed struct {
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	URL         string          `json:"url,omitempty"`
	Color       int             `json:"color,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Fields      []EmbedField    `json:"fields,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
}

type EmbedThumbnail struct {
	URL string `json:"url"`
}

type EmbedImage struct {
	URL string `json:"url"`
}

type WebhookPayload struct {
	Content   string  `json:"content,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

type MessageOptions struct {
	Type        MessageType
	Level       MessageLevel
	Title       string
	Description string
	Fields      []EmbedField
	Footer      *EmbedFooter
	Author      *EmbedAuthor
	Thumbnail   *EmbedThumbnail
	Image       *EmbedImage
	Username    string
	AvatarURL   string
	Timestamp   time.Time
}
