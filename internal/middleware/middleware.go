package middleware

import (
	"strings"

	"smap-api/pkg/response"
	"smap-api/pkg/scope"

	"github.com/gin-gonic/gin"
)

// Auth returns a middleware that validates JWT tokens and sets the payload in context.
// It extracts the token from the Authorization header and verifies it using the JWT manager.
func (m Middleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.l.Warnf(c.Request.Context(), "Missing Authorization header | Path: %s", c.Request.URL.Path)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			m.l.Warnf(c.Request.Context(), "Invalid Authorization header format | Path: %s", c.Request.URL.Path)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(authHeader[len(bearerPrefix):])
		if tokenString == "" {
			m.l.Warnf(c.Request.Context(), "Empty token in Authorization header | Path: %s", c.Request.URL.Path)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		payload, err := m.jwtManager.Verify(tokenString)
		if err != nil {
			m.l.Warnf(c.Request.Context(), "Token verification failed: %v | Path: %s", err, c.Request.URL.Path)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// Set payload in context for use in handlers
		ctx := c.Request.Context()
		ctx = scope.SetPayloadToContext(ctx, payload)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
