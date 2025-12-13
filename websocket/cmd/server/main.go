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
	redisSubscriber "smap-websocket/internal/redis"
	"smap-websocket/internal/server"
	ws "smap-websocket/internal/websocket"
	"smap-websocket/pkg/discord"
	"smap-websocket/pkg/jwt"
	"smap-websocket/pkg/log"
	"smap-websocket/pkg/redis"
)

// @title       WebSocket Service
// @description WebSocket notification hub service for real-time messaging
// @version     1.0
// @host        localhost:8081
// @schemes     ws http
// @BasePath    /
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config:", err)
		return
	}

	// Initialize logger
	logger := log.Init(log.ZapConfig{
		Level:    cfg.Logger.Level,
		Mode:     cfg.Logger.Mode,
		Encoding: cfg.Logger.Encoding,
	})

	ctx := context.Background()
	logger.Info(ctx, "Starting WebSocket Service...")

	// Initialize Discord webhook (optional)
	var discordClient *discord.Discord
	if cfg.Discord.WebhookID != "" && cfg.Discord.WebhookToken != "" {
		discordClient, err = discord.New(logger, &discord.DiscordWebhook{
			ID:    cfg.Discord.WebhookID,
			Token: cfg.Discord.WebhookToken,
		})
		if err != nil {
			logger.Warnf(ctx, "Failed to initialize Discord webhook: %v", err)
		} else {
			logger.Info(ctx, "Discord webhook initialized")
		}
	}

	// Initialize Redis client
	redisClient, err := redis.NewClient(redis.Config{
		Host:            cfg.Redis.Host,
		Password:        cfg.Redis.Password,
		DB:              cfg.Redis.DB,
		UseTLS:          cfg.Redis.UseTLS,
		MaxRetries:      cfg.Redis.MaxRetries,
		MinIdleConns:    cfg.Redis.MinIdleConns,
		PoolSize:        cfg.Redis.PoolSize,
		PoolTimeout:     cfg.Redis.PoolTimeout,
		ConnMaxIdleTime: cfg.Redis.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.Redis.ConnMaxLifetime,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Redis: %v", err)
		return
	}
	defer redisClient.Close()
	logger.Infof(ctx, "Redis connected successfully to %s", cfg.Redis.Host)

	// Initialize JWT validator
	jwtValidator := jwt.NewValidator(jwt.Config{
		SecretKey: cfg.JWT.SecretKey,
	})

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

	logger.Info(ctx, "Server shutdown complete")
}
