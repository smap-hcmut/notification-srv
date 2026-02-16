package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"smap-websocket/config"
	configRedis "smap-websocket/config/redis"
	redisSubscriber "smap-websocket/internal/redis"
	"smap-websocket/internal/server"
	ws "smap-websocket/internal/websocket"
	"smap-websocket/pkg/discord"
	"smap-websocket/pkg/jwt"
	"smap-websocket/pkg/log"
)

// @title       SMAP Notification Service
// @description SMAP Notification Service - WebSocket server for real-time notifications
// @version     1.0
// @host        localhost:8081
// @schemes     ws http
// @BasePath    /
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name smap_auth_token
// @description Authentication token stored in HttpOnly cookie
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Bearer token authentication. Format: "Bearer {token}"
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

	logger.Info(ctx, "Starting Notification WebSocket Service...")

	// Initialize Discord webhook (optional)
	var discordClient *discord.Discord
	if cfg.Discord.WebhookID != "" && cfg.Discord.WebhookToken != "" {
		webhook, err := discord.NewDiscordWebhook(cfg.Discord.WebhookID, cfg.Discord.WebhookToken)
		if err != nil {
			logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		} else {
			discordClient, err = discord.New(logger, webhook)
			if err != nil {
				logger.Warnf(ctx, "Failed to initialize Discord webhook: %v", err)
			} else {
				logger.Info(ctx, "Discord webhook initialized")
			}
		}
	}

	// Redis - Pub/Sub for real-time notifications
	redisClient, err := configRedis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Redis: %v", err)
		return
	}
	defer configRedis.Disconnect()
	logger.Infof(ctx, "Redis client initialized")

	// JWT Manager (verify tokens from cookie/header)
	jwtValidator := jwt.NewValidator(jwt.Config{
		SecretKey: cfg.JWT.SecretKey,
	})
	logger.Info(ctx, "JWT validator initialized")

	// Initialize WebSocket Hub
	hub := ws.NewHub(logger, cfg.WebSocket.MaxConnections)
	go hub.Run()
	logger.Info(ctx, "WebSocket Hub started")

	// Initialize Redis Subscriber
	subscriber := redisSubscriber.NewSubscriber(redisClient, hub, logger)
	if err := subscriber.Start(); err != nil {
		logger.Errorf(ctx, "Failed to start Redis subscriber: %v", err)
		return
	}
	logger.Info(ctx, "Redis Pub/Sub subscriber started")

	// Wire subscriber as notifier for Hub disconnect callbacks
	hub.SetRedisNotifier(subscriber)

	// Initialize WebSocket handler
	wsHandler := ws.NewHandler(
		hub,
		jwtValidator,
		logger,
		ws.WSConfig{
			PongWait:       cfg.WebSocket.PongWait,
			PingPeriod:     cfg.WebSocket.PingInterval,
			WriteWait:      cfg.WebSocket.WriteWait,
			MaxMessageSize: cfg.WebSocket.MaxMessageSize,
		},
		subscriber, // Implements RedisNotifier interface
		ws.CookieConfig{
			Domain:         cfg.Cookie.Domain,
			Secure:         cfg.Cookie.Secure,
			SameSite:       cfg.Cookie.SameSite,
			MaxAge:         cfg.Cookie.MaxAge,
			MaxAgeRemember: cfg.Cookie.MaxAgeRemember,
			Name:           cfg.Cookie.Name,
		},
		cfg.Environment.Name, // Pass environment for CORS configuration
	)

	// Setup Gin router
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	// Setup WebSocket routes
	wsHandler.SetupRoutes(router)

	// Setup server
	srv := server.New(server.Config{
		Host:          cfg.Server.Host,
		Port:          cfg.Server.Port,
		Router:        router,
		Logger:        logger,
		Hub:           hub,
		RedisClient:   redisClient,
		DiscordClient: discordClient,
	})

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			logger.Errorf(ctx, "Server error: %v", err)
		}
	}()

	logger.Infof(ctx, "WebSocket server listening on %s:%d", cfg.Server.Host, cfg.Server.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down gracefully...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown components in order
	if err := subscriber.Shutdown(shutdownCtx); err != nil {
		logger.Errorf(ctx, "Error shutting down Redis subscriber: %v", err)
	}

	if err := hub.Shutdown(shutdownCtx); err != nil {
		logger.Errorf(ctx, "Error shutting down Hub: %v", err)
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf(ctx, "Error shutting down server: %v", err)
	}

	logger.Info(ctx, "WebSocket server stopped gracefully")
}
