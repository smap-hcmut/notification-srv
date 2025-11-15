package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is a list of origins that are allowed to make requests.
	// Use "*" to allow all origins (not recommended for production).
	AllowedOrigins []string

	// AllowedMethods is a list of HTTP methods that are allowed.
	AllowedMethods []string

	// AllowedHeaders is a list of HTTP headers that are allowed.
	AllowedHeaders []string

	// ExposedHeaders is a list of headers that clients are allowed to access.
	ExposedHeaders []string

	// AllowCredentials indicates whether the request can include user credentials.
	AllowCredentials bool

	// MaxAge indicates how long (in seconds) the results of a preflight request can be cached.
	MaxAge int
}

// DefaultCORSConfig returns a default CORS configuration suitable for most APIs.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"Accept",
			"X-Requested-With",
			"lang",
		},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS).
// It sets appropriate CORS headers based on the provided configuration.
func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin, config.AllowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(config.AllowedOrigins) > 0 && config.AllowedOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if len(config.AllowedMethods) > 0 {
				c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			}
			if len(config.AllowedHeaders) > 0 {
				c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}
			if len(config.ExposedHeaders) > 0 {
				c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}
			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}
			c.AbortWithStatus(204) // No Content
			return
		}

		// Set headers for actual requests
		if len(config.ExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
		}
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Next()
	}
}

// isOriginAllowed checks if the given origin is in the allowed origins list.
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains (e.g., "*.example.com")
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
