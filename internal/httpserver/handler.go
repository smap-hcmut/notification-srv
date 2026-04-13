package httpserver

import (
	"context"
	alertUC "notification-srv/internal/alert/usecase"
	"notification-srv/internal/model"
	wsHTTP "notification-srv/internal/websocket/delivery/http"
	wsRedis "notification-srv/internal/websocket/delivery/redis"
	wsUC "notification-srv/internal/websocket/usecase"

	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// mapHandlers initializes and maps all HTTP routes
func (srv *HTTPServer) mapHandlers() error {
	// Initialize middleware
	mw := middleware.New(middleware.Config{
		JWTManager:       srv.jwtMgr,
		CookieName:       srv.cookieCfg.Name,
		ProductionDomain: srv.cookieCfg.Domain,
		InternalKey:      srv.internalKey,
		IsProduction:     srv.environment == string(model.EnvironmentProduction),
	})

	// Register middlewares
	srv.registerMiddlewares()

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
		srv.jwtMgr, // No assertion needed, srv.jwtMgr is auth.Manager
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
			Secure:   true, // Always secure for WebSocket (production-safe)
			HttpOnly: true,
			MaxAge:   srv.cookieCfg.MaxAge,
		},
		srv.environment,
	)

	// Register Routes
	// WebSocket is registered at root level (not under api/v1) because
	// Traefik strips /notification prefix → client calls /notification/ws → service receives /ws
	wsHandler.RegisterRoutes(srv.gin.Group(""), mw)

	return nil
}

// registerMiddlewares registers global middlewares
func (srv *HTTPServer) registerMiddlewares() {
	srv.gin.Use(middleware.Tracing())
	srv.gin.Use(middleware.Recovery(srv.logger, srv.discord))

	// CORS configuration based on environment
	corsConfig := middleware.DefaultCORSConfig(srv.environment)
	srv.gin.Use(middleware.CORS(corsConfig))

	// Log CORS mode for visibility
	ctx := context.Background()
	if srv.environment == string(model.EnvironmentProduction) {
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
