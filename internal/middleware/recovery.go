package middleware

import (
	"smap-api/pkg/discord"
	"smap-api/pkg/log"
	"smap-api/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery returns a middleware that recovers from panics and logs the error.
// It uses structured logging and reports panics to Discord if configured.
func Recovery(logger log.Logger, discordClient *discord.Discord) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx := c.Request.Context()
				logger.Errorf(ctx, "Panic recovered: %v | Method: %s | Path: %s",
					err, c.Request.Method, c.Request.URL.Path)

				response.PanicError(c, err, discordClient)
				c.Abort()
			}
		}()
		c.Next()
	}
}
