package middleware

import (
	"notification-srv/config"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/scope"
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
