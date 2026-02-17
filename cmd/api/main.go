package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"notification-srv/config"
	"notification-srv/config/redis"
	"notification-srv/internal/httpserver"
	"notification-srv/pkg/discord"
	"notification-srv/pkg/scope"
	"notification-srv/pkg/log"
)

// @title       SMAP Notification Service API
// @description SMAP Notification Service API documentation.
// @version     1
// @host        localhost:8080
// @schemes     http
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name smap_auth_token
// @description Authentication token stored in HttpOnly cookie. Set automatically by /login endpoint.
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Legacy Bearer token authentication (deprecated - use cookie authentication instead). Format: "Bearer {token}"
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config:", err)
		return
	}

	// Initialize logger
	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	// Create context with signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Redis - Pub/Sub for real-time notifications
	redisClient, err := redis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Redis: %v", err)
		return
	}
	defer redis.Disconnect()
	logger.Infof(ctx, "Redis client initialized")

	// Scope/JWT Manager (verify tokens from HttpOnly cookie)
	jwtManager := scope.New(cfg.JWT.SecretKey)
	logger.Infof(ctx, "Scope/JWT Manager initialized")

	// Discord - Monitoring & Notification
	discordClient, err := discord.New(logger, cfg.Discord.WebhookURL)
	if err != nil {
		logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		discordClient = nil
	} else {
		logger.Info(ctx, "Discord client initialized")
	}

	// HTTP server
	httpServer, err := httpserver.New(logger, httpserver.Config{
		// Server configuration
		Port:        cfg.Server.Port,
		Environment: cfg.Environment.Name,

		// WebSocket configuration
		WSConfig: cfg.WebSocket,

		// Auth & security
		JWTManager: jwtManager,
		Cookie:     cfg.Cookie,

		// External services
		Redis:   redisClient,
		Discord: discordClient,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize HTTP server: ", err)
		return
	}

	if err := httpServer.Run(); err != nil {
		logger.Error(ctx, "Failed to run server: ", err)
		return
	}

	logger.Info(ctx, "API server stopped gracefully")
}
