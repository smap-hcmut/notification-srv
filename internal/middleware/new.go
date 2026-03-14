package middleware

import (
	"notification-srv/config"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// Middleware wraps shared-libs auth.Middleware for backward compatibility
type Middleware struct {
	authMiddleware *auth.Middleware
	logger         log.Logger
}

// New creates a new middleware using shared-libs auth.Middleware
func New(logger log.Logger, jwtManager auth.Manager, cookieConfig config.CookieConfig) Middleware {
	authMiddleware := auth.NewMiddleware(auth.MiddlewareConfig{
		Manager:                 jwtManager,
		CookieName:              cookieConfig.Name,
		AllowBearerInProduction: os.Getenv("ENVIRONMENT_NAME") != "production",
		ProductionDomain:        cookieConfig.Domain,
	})

	return Middleware{
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// Auth returns the Gin authentication middleware
func (m Middleware) Auth() gin.HandlerFunc {
	return m.authMiddleware.Authenticate()
}
