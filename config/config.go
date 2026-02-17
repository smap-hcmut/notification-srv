package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all service configuration.
type Config struct {
	// Environment Configuration
	Environment EnvironmentConfig

	// Server Configuration
	Server ServerConfig
	Logger LoggerConfig

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

// EnvironmentConfig is the configuration for the deployment environment.
type EnvironmentConfig struct {
	Name string
}

// ServerConfig is the configuration for the WebSocket server
type ServerConfig struct {
	Port int
	Mode string
}

// RedisConfig is the configuration for Redis
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// WebSocketConfig is the configuration for WebSocket connections
type WebSocketConfig struct {
	PingInterval    time.Duration
	PongWait        time.Duration
	WriteWait       time.Duration
	MaxMessageSize  int64
	ReadBufferSize  int
	WriteBufferSize int
	MaxConnections  int
}

// JWTConfig is the configuration for the JWT
type JWTConfig struct {
	SecretKey string
}

// CookieConfig is the configuration for HttpOnly cookie authentication
type CookieConfig struct {
	Domain         string
	Secure         bool
	SameSite       string
	MaxAge         int
	MaxAgeRemember int
	Name           string
}

// LoggerConfig is the configuration for the logger
type LoggerConfig struct {
	Level        string
	Mode         string
	Encoding     string
	ColorEnabled bool
}

// DiscordConfig is the configuration for Discord webhook notifications
type DiscordConfig struct {
	WebhookURL string
}

// Load loads configuration using Viper
func Load() (*Config, error) {
	// Set config file name and paths
	viper.SetConfigName("notification-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/smap/")

	// Enable environment variable override
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := bindEnv(); err != nil {
		return nil, fmt.Errorf("error binding env vars: %w", err)
	}

	// Set defaults
	setDefaults()

	// Read config file (optional - will use env vars if file not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; using environment variables
	}

	cfg := &Config{}

	// Environment
	cfg.Environment.Name = viper.GetString("environment.name")

	// Server
	cfg.Server.Port = viper.GetInt("server.port")
	cfg.Server.Mode = viper.GetString("server.mode")

	// Logger
	cfg.Logger.Level = viper.GetString("logger.level")
	cfg.Logger.Mode = viper.GetString("logger.mode")
	cfg.Logger.Encoding = viper.GetString("logger.encoding")
	cfg.Logger.ColorEnabled = viper.GetBool("logger.color_enabled")

	// Redis
	cfg.Redis.Host = viper.GetString("redis.host")
	cfg.Redis.Port = viper.GetInt("redis.port")
	cfg.Redis.Password = viper.GetString("redis.password")
	cfg.Redis.DB = viper.GetInt("redis.db")

	// WebSocket
	cfg.WebSocket.PingInterval = viper.GetDuration("websocket.ping_interval")
	cfg.WebSocket.PongWait = viper.GetDuration("websocket.pong_wait")
	cfg.WebSocket.WriteWait = viper.GetDuration("websocket.write_wait")
	cfg.WebSocket.MaxMessageSize = viper.GetInt64("websocket.max_message_size")
	cfg.WebSocket.ReadBufferSize = viper.GetInt("websocket.read_buffer_size")
	cfg.WebSocket.WriteBufferSize = viper.GetInt("websocket.write_buffer_size")
	cfg.WebSocket.MaxConnections = viper.GetInt("websocket.max_connections")

	// JWT
	cfg.JWT.SecretKey = viper.GetString("jwt.secret_key")

	// Cookie
	cfg.Cookie.Domain = viper.GetString("cookie.domain")
	cfg.Cookie.Secure = viper.GetBool("cookie.secure")
	cfg.Cookie.SameSite = viper.GetString("cookie.samesite")
	cfg.Cookie.MaxAge = viper.GetInt("cookie.max_age")
	cfg.Cookie.MaxAgeRemember = viper.GetInt("cookie.max_age_remember")
	cfg.Cookie.Name = viper.GetString("cookie.name")

	// Discord
	cfg.Discord.WebhookURL = viper.GetString("discord.webhook_url")

	// Validate required fields
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults() {
	// Environment
	viper.SetDefault("environment.name", "production")

	// Server
	viper.SetDefault("server.port", 8081)
	viper.SetDefault("server.mode", "release")

	// Logger
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.mode", "production")
	viper.SetDefault("logger.encoding", "json")
	viper.SetDefault("logger.color_enabled", false)

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// WebSocket
	viper.SetDefault("websocket.ping_interval", 30*time.Second)
	viper.SetDefault("websocket.pong_wait", 60*time.Second)
	viper.SetDefault("websocket.write_wait", 10*time.Second)
	viper.SetDefault("websocket.max_message_size", 512)
	viper.SetDefault("websocket.read_buffer_size", 1024)
	viper.SetDefault("websocket.write_buffer_size", 1024)
	viper.SetDefault("websocket.max_connections", 10000)

	// Cookie
	viper.SetDefault("cookie.domain", ".smap.com")
	viper.SetDefault("cookie.secure", true)
	viper.SetDefault("cookie.samesite", "Lax")
	viper.SetDefault("cookie.max_age", 7200)
	viper.SetDefault("cookie.max_age_remember", 2592000)
	viper.SetDefault("cookie.name", "smap_auth_token")

	// Discord (optional)
	viper.SetDefault("discord.webhook_url", "")
}

func validate(cfg *Config) error {
	// Validate JWT
	if cfg.JWT.SecretKey == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if len(cfg.JWT.SecretKey) < 32 {
		return fmt.Errorf("jwt.secret_key must be at least 32 characters for security")
	}

	// Validate Server
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port is invalid")
	}

	// Validate Redis
	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if cfg.Redis.Port == 0 {
		return fmt.Errorf("redis.port is required")
	}

	// Validate Cookie
	if cfg.Cookie.Name == "" {
		return fmt.Errorf("cookie.name is required")
	}

	return nil
}

func bindEnv() error {
	// Support both canonical env var names (SERVER_PORT, WEBSOCKET_*, ...)
	// and legacy names used in some manifests (WS_*, ENV).
	binds := map[string][]string{
		"environment.name": {"ENVIRONMENT_NAME", "ENV"},

		"server.port": {"SERVER_PORT", "WS_PORT"},
		"server.mode": {"SERVER_MODE", "WS_MODE"},

		"logger.level":         {"LOGGER_LEVEL"},
		"logger.mode":          {"LOGGER_MODE"},
		"logger.encoding":      {"LOGGER_ENCODING"},
		"logger.color_enabled": {"LOGGER_COLOR_ENABLED"},

		"redis.host":     {"REDIS_HOST"},
		"redis.port":     {"REDIS_PORT"},
		"redis.password": {"REDIS_PASSWORD"},
		"redis.db":       {"REDIS_DB"},

		"websocket.ping_interval":     {"WEBSOCKET_PING_INTERVAL", "WS_PING_INTERVAL"},
		"websocket.pong_wait":         {"WEBSOCKET_PONG_WAIT", "WS_PONG_WAIT"},
		"websocket.write_wait":        {"WEBSOCKET_WRITE_WAIT", "WS_WRITE_WAIT"},
		"websocket.max_message_size":  {"WEBSOCKET_MAX_MESSAGE_SIZE", "WS_MAX_MESSAGE_SIZE"},
		"websocket.read_buffer_size":  {"WEBSOCKET_READ_BUFFER_SIZE", "WS_READ_BUFFER_SIZE"},
		"websocket.write_buffer_size": {"WEBSOCKET_WRITE_BUFFER_SIZE", "WS_WRITE_BUFFER_SIZE"},
		"websocket.max_connections":   {"WEBSOCKET_MAX_CONNECTIONS", "WS_MAX_CONNECTIONS"},

		"jwt.secret_key": {"JWT_SECRET_KEY"},

		"cookie.domain":           {"COOKIE_DOMAIN"},
		"cookie.secure":           {"COOKIE_SECURE"},
		"cookie.samesite":         {"COOKIE_SAMESITE"},
		"cookie.max_age":          {"COOKIE_MAX_AGE"},
		"cookie.max_age_remember": {"COOKIE_MAX_AGE_REMEMBER"},
		"cookie.name":             {"COOKIE_NAME"},

		"discord.webhook_url": {"DISCORD_WEBHOOK_URL"},
	}

	for key, envs := range binds {
		args := append([]string{key}, envs...)
		if err := viper.BindEnv(args...); err != nil {
			return fmt.Errorf("bind %s: %w", key, err)
		}
	}
	return nil
}
