package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	// Server Configuration
	Server ServerConfig
	Logger LoggerConfig

	// Environment Configuration
	Environment EnvironmentConfig

	// Redis Configuration
	Redis RedisConfig

	// WebSocket Configuration
	WebSocket WebSocketConfig

	// Authentication & Security Configuration
	JWT    JWTConfig
	Cookie CookieConfig

	// Monitoring & Notification Configuration
	Discord DiscordConfig
}

// ServerConfig is the configuration for the WebSocket server
type ServerConfig struct {
	Host string `env:"WS_HOST" envDefault:"0.0.0.0"`
	Port int    `env:"WS_PORT" envDefault:"8081"`
	Mode string `env:"WS_MODE" envDefault:"release"`
}

// RedisConfig is the configuration for Redis
// Note: Only standalone mode is supported
type RedisConfig struct {
	Host     string `env:"REDIS_HOST" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
	UseTLS   bool   `env:"REDIS_USE_TLS" envDefault:"false"`

	// Connection pool settings
	MaxRetries      int           `env:"REDIS_MAX_RETRIES" envDefault:"3"`
	MinIdleConns    int           `env:"REDIS_MIN_IDLE_CONNS" envDefault:"10"`
	PoolSize        int           `env:"REDIS_POOL_SIZE" envDefault:"100"`
	PoolTimeout     time.Duration `env:"REDIS_POOL_TIMEOUT" envDefault:"4s"`
	ConnMaxIdleTime time.Duration `env:"REDIS_CONN_MAX_IDLE_TIME" envDefault:"5m"`
	ConnMaxLifetime time.Duration `env:"REDIS_CONN_MAX_LIFETIME" envDefault:"30m"`
}

// WebSocketConfig is the configuration for WebSocket connections
type WebSocketConfig struct {
	PingInterval    time.Duration `env:"WS_PING_INTERVAL" envDefault:"30s"`
	PongWait        time.Duration `env:"WS_PONG_WAIT" envDefault:"60s"`
	WriteWait       time.Duration `env:"WS_WRITE_WAIT" envDefault:"10s"`
	MaxMessageSize  int64         `env:"WS_MAX_MESSAGE_SIZE" envDefault:"512"`
	ReadBufferSize  int           `env:"WS_READ_BUFFER_SIZE" envDefault:"1024"`
	WriteBufferSize int           `env:"WS_WRITE_BUFFER_SIZE" envDefault:"1024"`
	MaxConnections  int           `env:"WS_MAX_CONNECTIONS" envDefault:"10000"`
}

// JWTConfig is the configuration for the JWT
type JWTConfig struct {
	SecretKey string `env:"JWT_SECRET_KEY"`
}

// CookieConfig is the configuration for HttpOnly cookie authentication
type CookieConfig struct {
	Domain         string `env:"COOKIE_DOMAIN" envDefault:".smap.com"`
	Secure         bool   `env:"COOKIE_SECURE" envDefault:"true"`
	SameSite       string `env:"COOKIE_SAMESITE" envDefault:"Lax"`
	MaxAge         int    `env:"COOKIE_MAX_AGE" envDefault:"7200"`
	MaxAgeRemember int    `env:"COOKIE_MAX_AGE_REMEMBER" envDefault:"2592000"`
	Name           string `env:"COOKIE_NAME" envDefault:"smap_auth_token"`
}

// LoggerConfig is the configuration for the logger
type LoggerConfig struct {
	Level        string `env:"LOGGER_LEVEL" envDefault:"info"`
	Mode         string `env:"LOGGER_MODE" envDefault:"production"`
	Encoding     string `env:"LOGGER_ENCODING" envDefault:"json"`
	ColorEnabled bool   `env:"LOGGER_COLOR_ENABLED" envDefault:"true"`
}

// EnvironmentConfig is the configuration for environment-aware features
type EnvironmentConfig struct {
	Name string `env:"ENV" envDefault:"production"`
}

// DiscordConfig is the configuration for Discord webhook notifications
type DiscordConfig struct {
	WebhookID    string `env:"DISCORD_WEBHOOK_ID"`
	WebhookToken string `env:"DISCORD_WEBHOOK_TOKEN"`
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		fmt.Printf("Error loading configuration: %v", err)
		return nil, err
	}
	return cfg, nil
}
