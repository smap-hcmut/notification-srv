package httpserver

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/log"
)

func requestLogger(l log.Logger, environment string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		if path == "/health" || path == "/ready" || path == "/live" {
			return
		}

		status := c.Writer.Status()
		if path == "/ws" && status == 401 {
			// Browsers with stale tabs can retry unauthenticated websocket handshakes.
			// Invalid tokens are still logged by the websocket handler itself.
			return
		}

		latency := time.Since(start)
		ctx := c.Request.Context()

		if environment == "production" {
			msg := "HTTP Request - Method: %s, Path: %s, Status: %d, IP: %s, Latency: %v, UserAgent: %s, Query: %s"
			args := []any{c.Request.Method, path, status, c.ClientIP(), latency, c.Request.UserAgent(), query}
			switch {
			case status >= 500:
				l.Errorf(ctx, msg, args...)
			case status >= 400:
				l.Warnf(ctx, msg, args...)
			default:
				l.Infof(ctx, msg, args...)
			}
			return
		}

		msg := "%s %s %d %s %s"
		args := []any{c.Request.Method, path, status, latency, c.ClientIP()}
		switch {
		case status >= 500:
			l.Errorf(ctx, msg, args...)
		case status >= 400:
			l.Warnf(ctx, msg, args...)
		default:
			l.Infof(ctx, msg, args...)
		}
	}
}
