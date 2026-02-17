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
//   1. Start Hub background service (message routing)
//   2. Map HTTP handlers and routes
//   3. Start HTTP server
//   4. Wait for shutdown signal
func (srv *HTTPServer) Run() error {
	ctx := context.Background()

	// 1. Start Hub background service for WebSocket message routing
	go srv.hub.Run()
	srv.logger.Info(ctx, "WebSocket Hub background service started")

	// 2. Map handlers (initializes Redis subscriber, WebSocket routes, etc.)
	if err := srv.mapHandlers(); err != nil {
		srv.logger.Fatalf(ctx, "Failed to map handlers: %v", err)
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

	return nil
}
