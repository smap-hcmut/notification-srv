package middleware

import (
	"smap-api/pkg/locale"

	"github.com/gin-gonic/gin"
)

// Locale returns a middleware that extracts and sets the locale from the request header.
// It reads the "lang" header, validates it, and sets the locale in the request context.
// If no valid locale is provided, it defaults to the system default locale.
func (m Middleware) Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		langHeader := c.GetHeader("lang")

		// Parse and validate the language header
		lang := locale.ParseLang(langHeader)

		// Set locale in context for use in handlers
		ctx := c.Request.Context()
		ctx = locale.SetLocaleToContext(ctx, lang)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
