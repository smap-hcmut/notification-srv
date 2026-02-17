package httpserver

import (
	"context"

	alertUC "notification-srv/internal/alert/usecase"
	"notification-srv/internal/middleware"
	wsHTTP "notification-srv/internal/websocket/delivery/http"
	wsRedis "notification-srv/internal/websocket/delivery/redis"
	wsUC "notification-srv/internal/websocket/usecase"
)

// mapHandlers initializes and maps all HTTP routes
func (srv *HTTPServer) mapHandlers() error {
	// Initialize middleware
	mw := middleware.New(srv.logger, srv.jwtMgr, srv.cookieCfg)

	// Register middlewares
	srv.registerMiddlewares(mw)

	// Register system routes (health checks)
	srv.registerSystemRoutes()

	// --- Domain Wiring ---

	// 1. Alert (Reference Domain)
	alertUseCase := alertUC.New(srv.logger, srv.discord)

	// 2. WebSocket Domain
	// UseCase
	srv.wsUC = wsUC.New(srv.logger, srv.wsConfig.MaxConnections, alertUseCase)

	// Delivery: Redis Subscriber
	srv.wsSubscriber = wsRedis.New(srv.redis, srv.wsUC, srv.logger)
	// Subscriber start is handled in Run()

	// Delivery: HTTP Handler
	wsHandler := wsHTTP.New(
		srv.wsUC,
		srv.jwtMgr, // No assertion needed, srv.jwtMgr is scope.Manager
		srv.logger,
		wsHTTP.WSConfig{
			MaxConnections:  srv.wsConfig.MaxConnections,
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			AllowedOrigins:  []string{"*"},
		},
		wsHTTP.CookieConfig{
			Name:     srv.cookieCfg.Name,
			Domain:   srv.cookieCfg.Domain,
			Path:     "/",
			Secure:   srv.cookieCfg.Secure,
			HttpOnly: true,
			MaxAge:   srv.cookieCfg.MaxAge,
		},
		srv.environment,
	)

	// Register Routes
	wsHandler.RegisterRoutes(srv.gin.Group(""), mw)

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
