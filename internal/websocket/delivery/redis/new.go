package redis

import (
	"context"
	"os"
	"sync"

	"notification-srv/internal/websocket"

	"github.com/smap-hcmut/shared-libs/go/log"
	pkgRedis "github.com/smap-hcmut/shared-libs/go/redis"
)

const (
	// notificationStream must match analyticsbridge.notificationStream so a
	// single fan-in carries everything that used to flow through the legacy
	// PubSub channels.
	notificationStream = "smap:notifications:stream"

	// consumerGroup partitions the stream across notification-srv replicas.
	// Same group name across pods = load-balanced consumption.
	consumerGroup = "notification-srv"

	// streamReadCount caps how many entries we pull per XReadGroup call.
	streamReadCount int64 = 32
)

type Subscriber interface {
	Start() error
	Shutdown(ctx context.Context) error
}

type subscriber struct {
	redis        pkgRedis.IRedis
	uc           websocket.UseCase
	logger       log.Logger
	consumerName string

	// Lifecycle fields
	wg   sync.WaitGroup
	quit chan struct{}
}

func New(redis pkgRedis.IRedis, uc websocket.UseCase, logger log.Logger) Subscriber {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "notification-srv"
	}
	return &subscriber{
		redis:        redis,
		uc:           uc,
		logger:       logger,
		consumerName: hostname,
		quit:         make(chan struct{}),
	}
}
