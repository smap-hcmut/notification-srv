package http

import (
	"notification-srv/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// Handler defines the HTTP handler interface for WebSocket.
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware)
}

type handler struct {
	uc          websocket.UseCase
	jwtMgr      auth.Manager
	logger      log.Logger
	wsConfig    WSConfig
	cookieCfg   CookieConfig
	environment string
}

func New(uc websocket.UseCase, jwtMgr auth.Manager, logger log.Logger, wsCfg WSConfig, cookieCfg CookieConfig, env string) Handler {
	return &handler{
		uc:          uc,
		jwtMgr:      jwtMgr,
		logger:      logger,
		wsConfig:    wsCfg,
		cookieCfg:   cookieCfg,
		environment: env,
	}
}
