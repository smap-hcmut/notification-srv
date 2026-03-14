package httpserver

import (
	"errors"

	"notification-srv/config"
	"notification-srv/internal/websocket"
	"notification-srv/internal/websocket/delivery/redis"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
	pkgRedis "github.com/smap-hcmut/shared-libs/go/redis"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

// HTTPServer represents the HTTP server with all dependencies.
// New() only wires dependencies and validates them.
// Run() (in httpserver.go) is responsible for starting background services and HTTP serving.
type HTTPServer struct {
	// Server configuration
	gin         *gin.Engine
	logger      log.Logger
	port        int
	environment string

	// WebSocket core (New Domain)
	wsUC         websocket.UseCase
	wsSubscriber redis.Subscriber
	wsConfig     config.WebSocketConfig

	// Auth & security
	jwtMgr    auth.Manager
	cookieCfg config.CookieConfig

	// External services
	redis   pkgRedis.IRedis
	discord discord.IDiscord
}

// Config is the constructor input for HTTPServer.
// Keep this minimal: only fields really needed by HTTPServer.
type Config struct {
	// Server configuration
	Port        int
	Mode        string
	Environment string

	// WebSocket configuration
	WSConfig config.WebSocketConfig

	// Auth & security
	JWTManager auth.Manager
	Cookie     config.CookieConfig

	// External services
	Redis   pkgRedis.IRedis
	Discord discord.IDiscord
}

// New creates a new HTTPServer instance with the provided configuration.
// Note: This does NOT start any goroutines. Use (*HTTPServer).Run() to start the service.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	// Map environment name to gin mode.
	// We only allow standard gin modes: debug, release, test.
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		// Server configuration
		gin:         gin.New(),
		logger:      logger,
		port:        cfg.Port,
		environment: cfg.Environment,

		// WebSocket config
		wsConfig: cfg.WSConfig,

		// Auth & security
		jwtMgr:    cfg.JWTManager,
		cookieCfg: cfg.Cookie,

		// External services
		redis:   cfg.Redis,
		discord: cfg.Discord,
	}

	// Add middlewares
	srv.gin.Use(srv.zapLoggerMiddleware())
	srv.gin.Use(gin.Recovery())

	if err := srv.validate(); err != nil {
		return nil, err
	}

	return srv, nil
}

func (srv *HTTPServer) zapLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if path == "/health" || path == "/ready" || path == "/live" {
			return
		}

		if srv.environment == "production" {
			srv.logger.Infof(c.Request.Context(),
				"HTTP Request - Method: %s, Path: %s, Status: %d, IP: %s, Latency: %v, UserAgent: %s, Query: %s",
				c.Request.Method, path, status, c.ClientIP(), latency, c.Request.UserAgent(), query)
		} else {
			srv.logger.Infof(c.Request.Context(), "%s %s %d %s %s", c.Request.Method, path, status, latency, c.ClientIP())
		}
	}
}

// validate ensures all required dependencies are provided.
func (s *HTTPServer) validate() error {
	if s.logger == nil {
		return errors.New("logger is required")
	}
	if s.port == 0 {
		return errors.New("port is required")
	}
	if s.jwtMgr == nil {
		return errors.New("JWTManager is required")
	}
	if s.redis == nil {
		return errors.New("Redis client is required")
	}

	return nil
}
