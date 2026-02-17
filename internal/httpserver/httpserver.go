package httpserver

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Run starts the HTTP server and all background services, then blocks until shutdown signal.
// This method manages the complete lifecycle of the WebSocket service:
//  1. Map HTTP handlers and routes (Initialize wiring)
//  2. Start WebSocket UseCase (Hub)
//  3. Start HTTP server
//  4. Wait for shutdown signal
func (srv *HTTPServer) Run() error {
	ctx := context.Background()

	// 1. Map handlers (initializes WebSocket UseCase, Subscriber, Routes)
	if err := srv.mapHandlers(); err != nil {
		srv.logger.Fatalf(ctx, "Failed to map handlers: %v", err)
		return err
	}

	// 2. Start WebSocket background services
	// Start UseCase (Hub)
	go srv.wsUC.Run()
	srv.logger.Info(ctx, "WebSocket UseCase background service started")

	// Start Redis Subscriber
	if err := srv.wsSubscriber.Start(); err != nil {
		srv.logger.Fatalf(ctx, "Failed to start Redis subscriber: %v", err)
		return err
	}

	// 3. Start HTTP server in background
	go func() {
		if err := srv.gin.Run(fmt.Sprintf(":%d", srv.port)); err != nil {
			srv.logger.Errorf(ctx, "HTTP server error: %v", err)
		}
	}()

	srv.logger.Infof(ctx, "HTTP server started on port: %d", srv.port)

	// 4. Wait for shutdown signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	srv.logger.Info(ctx, <-ch)
	srv.logger.Info(ctx, "Stopping WebSocket service...")

	// Graceful shutdown
	if err := srv.wsUC.Shutdown(ctx); err != nil {
		srv.logger.Errorf(ctx, "WebSocket UseCase shutdown error: %v", err)
	}
	if err := srv.wsSubscriber.Shutdown(ctx); err != nil {
		srv.logger.Errorf(ctx, "Redis Subscriber shutdown error: %v", err)
	}

	return nil
}
