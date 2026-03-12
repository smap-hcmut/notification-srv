package http

import (
	"notification-srv/internal/websocket"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/scope"
)

type Handler struct {
	uc          websocket.UseCase
	jwtMgr      scope.Manager
	logger      log.Logger
	wsConfig    WSConfig
	cookieCfg   CookieConfig
	environment string
}

func New(uc websocket.UseCase, jwtMgr scope.Manager, logger log.Logger, wsCfg WSConfig, cookieCfg CookieConfig, env string) *Handler {
	return &Handler{
		uc:          uc,
		jwtMgr:      jwtMgr,
		logger:      logger,
		wsConfig:    wsCfg,
		cookieCfg:   cookieCfg,
		environment: env,
	}
}
