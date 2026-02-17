package redis

import (
	"context"

	"notification-srv/internal/websocket"

	"github.com/redis/go-redis/v9"
)

func (s *subscriber) handleMessage(ctx context.Context, msg *redis.Message) {
	input := websocket.ProcessMessageInput{
		Channel: msg.Channel,
		Payload: []byte(msg.Payload),
	}

	if err := s.uc.ProcessMessage(ctx, input); err != nil {
		s.logger.Warnf(ctx, "process message failed: channel=%s err=%v", msg.Channel, err)
	}
}
