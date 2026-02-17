package httpserver

import (
	"context"

	"notification-srv/internal/middleware"
	redisSubscriber "notification-srv/internal/websocket/delivery/redis"
	wsHTTP "notification-srv/internal/websocket/delivery/http"
	"notification-srv/pkg/scope"
)

// mapHandlers initializes and maps all HTTP routes
func (srv *HTTPServer) mapHandlers() error {
	// Initialize middleware
	mw := middleware.New(srv.logger, srv.jwtMgr, srv.cookieCfg)

	// Register middlewares
	srv.registerMiddlewares(mw)

	// Register system routes (health checks)
	srv.registerSystemRoutes()

	// Initialize Redis subscriber (for WebSocket message routing)
	subscriber := redisSubscriber.NewSubscriber(srv.redis, srv.hub, srv.logger)
	if err := subscriber.Start(); err != nil {
		return err
	}

	// Wire subscriber as notifier for Hub disconnect callbacks
	srv.hub.SetRedisNotifier(subscriber)

	// Initialize WebSocket handler (adapt scope.Manager to wsHTTP.JWTValidator)
	jwtValidator := jwtValidatorAdapter{mgr: srv.jwtMgr}
	srv.wsHandler = wsHTTP.NewHandler(
		srv.hub,
		jwtValidator,
		srv.logger,
		wsHTTP.WSConfig{
			PongWait:       srv.wsConfig.PongWait,
			PingPeriod:     srv.wsConfig.PingInterval,
			WriteWait:      srv.wsConfig.WriteWait,
			MaxMessageSize: srv.wsConfig.MaxMessageSize,
		},
		subscriber,
		wsHTTP.CookieConfig{
			Domain:         srv.cookieCfg.Domain,
			Secure:         srv.cookieCfg.Secure,
			SameSite:       srv.cookieCfg.SameSite,
			MaxAge:         srv.cookieCfg.MaxAge,
			MaxAgeRemember: srv.cookieCfg.MaxAgeRemember,
			Name:           srv.cookieCfg.Name,
		},
		srv.environment,
	)

	// Map WebSocket routes
	srv.wsHandler.SetupRoutes(srv.gin)

	return nil
}

// registerMiddlewares registers global middlewares
func (srv *HTTPServer) registerMiddlewares(mw middleware.Middleware) {
	srv.gin.Use(middleware.Recovery(srv.logger, srv.discord))

	// CORS configuration based on environment
	corsConfig := middleware.DefaultCORSConfig(srv.environment)
	srv.gin.Use(middleware.CORS(corsConfig))

	// Log CORS mode for visibility
	ctx := context.Background()
	if srv.environment == "production" {
		srv.logger.Infof(ctx, "CORS mode: production (strict origins only)")
	} else {
		srv.logger.Infof(ctx, "CORS mode: %s (permissive - allows localhost and private subnets)", srv.environment)
	}
}

// registerSystemRoutes registers health check and monitoring routes
func (srv *HTTPServer) registerSystemRoutes() {
	srv.gin.GET("/health", srv.healthCheck)
	srv.gin.GET("/ready", srv.readyCheck)
	srv.gin.GET("/live", srv.liveCheck)
}

// jwtValidatorAdapter adapts scope.Manager to wsHTTP.JWTValidator interface
type jwtValidatorAdapter struct {
	mgr scope.Manager
}

func (v jwtValidatorAdapter) ExtractUserID(tokenString string) (string, error) {
	payload, err := v.mgr.Verify(tokenString)
	if err != nil {
		return "", err
	}
	// Try UserID first, fallback to Subject
	if payload.UserID != "" {
		return payload.UserID, nil
	}
	return payload.Subject, nil
}
