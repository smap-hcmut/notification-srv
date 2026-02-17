package http

import (
	"notification-srv/internal/websocket"
	"notification-srv/pkg/log"
	"notification-srv/pkg/scope"
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
