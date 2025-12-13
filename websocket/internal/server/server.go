package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	ws "smap-websocket/internal/websocket"
	"smap-websocket/pkg/discord"
	"smap-websocket/pkg/log"
	"smap-websocket/pkg/redis"
)

// Server represents the HTTP server
type Server struct {
	config Config
	server *http.Server
}

// Config holds server configuration
type Config struct {
	Host          string
	Port          int
	Router        *gin.Engine
	Logger        log.Logger
	Hub           *ws.Hub
	RedisClient   *redis.Client
	DiscordClient *discord.Discord
}

// New creates a new Server instance
func New(cfg Config) *Server {
	// Setup routes
	setupRoutes(cfg.Router, cfg.Logger, cfg.Hub, cfg.RedisClient)

	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:        cfg.Router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return &Server{
		config: cfg,
		server: server,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.config.Logger.Infof(context.Background(), "Starting HTTP server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.config.Logger.Info(ctx, "Shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}

// setupRoutes sets up HTTP routes
func setupRoutes(router *gin.Engine, logger log.Logger, hub *ws.Hub, redisClient *redis.Client) {
	// Basic health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "websocket-service",
		})
	})
}
