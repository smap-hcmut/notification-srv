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
	Host string
	Port int
	Mode string
}

// RedisConfig is the configuration for Redis
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	UseTLS   bool

	// Connection pool settings
	MaxRetries      int
	MinIdleConns    int
	PoolSize        int
	PoolTimeout     time.Duration
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
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
	WebhookID    string
	WebhookToken string
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
	cfg.Server.Host = viper.GetString("server.host")
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
	cfg.Redis.UseTLS = viper.GetBool("redis.use_tls")
	cfg.Redis.MaxRetries = viper.GetInt("redis.max_retries")
	cfg.Redis.MinIdleConns = viper.GetInt("redis.min_idle_conns")
	cfg.Redis.PoolSize = viper.GetInt("redis.pool_size")
	cfg.Redis.PoolTimeout = viper.GetDuration("redis.pool_timeout")
	cfg.Redis.ConnMaxIdleTime = viper.GetDuration("redis.conn_max_idle_time")
	cfg.Redis.ConnMaxLifetime = viper.GetDuration("redis.conn_max_lifetime")

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
	cfg.Discord.WebhookID = viper.GetString("discord.webhook_id")
	cfg.Discord.WebhookToken = viper.GetString("discord.webhook_token")

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
	viper.SetDefault("server.host", "0.0.0.0")
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
	viper.SetDefault("redis.use_tls", false)
	viper.SetDefault("redis.max_retries", 3)
	viper.SetDefault("redis.min_idle_conns", 10)
	viper.SetDefault("redis.pool_size", 100)
	viper.SetDefault("redis.pool_timeout", 4*time.Second)
	viper.SetDefault("redis.conn_max_idle_time", 5*time.Minute)
	viper.SetDefault("redis.conn_max_lifetime", 30*time.Minute)

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
}

func validate(cfg *Config) error {
	// Validate JWT
	if cfg.JWT.SecretKey == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if len(cfg.JWT.SecretKey) < 32 {
		return fmt.Errorf("jwt.secret_key must be at least 32 characters for security")
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
