package middleware

import (
	"notification-srv/pkg/locale"

	"github.com/gin-gonic/gin"
)

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
