package redis

import (
	"context"
	"sync"

	"notification-srv/internal/websocket"

	"github.com/smap-hcmut/shared-libs/go/log"
	pkgRedis "github.com/smap-hcmut/shared-libs/go/redis"

	"github.com/redis/go-redis/v9"
)

type Subscriber interface {
	Start() error
	Shutdown(ctx context.Context) error
}

type subscriber struct {
	redis  pkgRedis.IRedis
	uc     websocket.UseCase
	logger log.Logger

	// Lifecycle fields
	pubsub *redis.PubSub
	wg     sync.WaitGroup
	quit   chan struct{}
}

func New(redis pkgRedis.IRedis, uc websocket.UseCase, logger log.Logger) Subscriber {
	return &subscriber{
		redis:  redis,
		uc:     uc,
		logger: logger,
		quit:   make(chan struct{}),
	}
}
