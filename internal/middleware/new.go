package middleware

import (
	"notification-srv/config"
	"notification-srv/pkg/log"
	"notification-srv/pkg/scope"
)

type Middleware struct {
	logger       log.Logger
	jwtManager   scope.Manager
	cookieConfig config.CookieConfig
}

func New(logger log.Logger, jwtManager scope.Manager, cookieConfig config.CookieConfig) Middleware {
	return Middleware{
		logger:       logger,
		jwtManager:   jwtManager,
		cookieConfig: cookieConfig,
	}
}
