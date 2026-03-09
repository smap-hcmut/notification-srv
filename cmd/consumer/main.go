package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"notification-srv/config"
	cfgRedis "notification-srv/config/redis"
	"notification-srv/pkg/discord"
	"notification-srv/pkg/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config:", err)
		return
	}

	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisClient, err := cfgRedis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Redis: %v", err)
		return
	}
	defer cfgRedis.Disconnect()
	logger.Info(ctx, "Redis client initialized")

	discordClient, err := discord.New(logger, cfg.Discord.WebhookURL)
	if err != nil {
		logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		discordClient = nil
	} else {
		logger.Info(ctx, "Discord client initialized")
	}

	// TODO: Initialize consumer-specific use cases and subscribers here
	_ = redisClient
	_ = discordClient

	logger.Info(ctx, "Consumer started, waiting for messages...")

	<-ctx.Done()
	logger.Info(ctx, "Consumer stopped gracefully")
}
