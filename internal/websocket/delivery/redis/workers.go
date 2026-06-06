package redis

import (
	"context"

	"notification-srv/internal/websocket"
)

// dispatch forwards a stream entry to the websocket usecase. Split out from
// the stream loop so unit tests can drive the same path without faking a
// Redis client.
func (s *subscriber) dispatch(ctx context.Context, channel string, payload []byte) {
	input := websocket.ProcessMessageInput{
		Channel: channel,
		Payload: payload,
	}

	if err := s.uc.ProcessMessage(ctx, input); err != nil {
		s.logger.Errorf(ctx, "process message failed: channel=%s err=%v", channel, err)
	}
}
