package httpserver

import (
	"smap-api/internal/middleware"

	// Import this to execute the init function in docs.go which setups the Swagger docs.
	_ "smap-api/docs" // TODO: Generate docs package

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	Api         = "/api/v1"
	InternalApi = "internal/api/v1"
)

func (srv HTTPServer) mapHandlers() error {
	// Apply CORS middleware globally
	corsConfig := middleware.DefaultCORSConfig()
	srv.gin.Use(middleware.CORS(corsConfig))

	// Health check endpoints (no auth required)
	srv.gin.GET("/health", srv.healthCheck)
	srv.gin.GET("/ready", srv.readyCheck)
	srv.gin.GET("/live", srv.liveCheck)

	// Swagger UI
	srv.gin.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	// api := srv.gin.Group(Api)
	// Apply auth middleware to protected routes
	// api.Use(middlewareInstance.Auth())

	return nil
}
