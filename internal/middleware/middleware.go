package middleware

import (
	"notification-srv/pkg/response"
	"notification-srv/pkg/scope"

	"github.com/gin-gonic/gin"
)

// Auth returns a middleware that authenticates requests using JWT.
// Priority:
//   1. Authorization header (Bearer token) - for dev/testing/mobile apps
//   2. HttpOnly cookie - for browser clients
func (m Middleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		var err error

		// Priority 1: Try to read token from Authorization header (for dev/testing/mobile)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Support both "Bearer <token>" and plain token
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString = authHeader[7:]
			} else {
				tokenString = authHeader
			}
		}

		// Priority 2: If no token in header, try cookie (for browser with HttpOnly)
		if tokenString == "" {
			tokenString, err = c.Cookie(m.cookieConfig.Name)
			if err != nil || tokenString == "" {
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		// Verify JWT token
		payload, err := m.jwtManager.Verify(tokenString)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// Set payload and scope in context for downstream handlers
		ctx := c.Request.Context()
		ctx = scope.SetPayloadToContext(ctx, payload)
		sc := scope.NewScope(payload)
		ctx = scope.SetScopeToContext(ctx, sc)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
