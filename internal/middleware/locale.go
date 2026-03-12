package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/locale"
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
