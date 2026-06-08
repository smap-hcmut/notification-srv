package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Start runs the consumer-group reader. Replaces the legacy PubSub
// PSubscribe so a slow WebSocket subscriber no longer drops notifications —
// every entry stays in the stream until the consumer XAcks it.
func (s *subscriber) Start() error {
	ctx := context.Background()

	if err := s.ensureGroup(ctx); err != nil {
		return fmt.Errorf("notification-srv: failed to ensure consumer group: %w", err)
	}

	s.wg.Add(1)
	go s.listen(ctx)

	s.logger.Infof(ctx, "Redis stream consumer started: stream=%s group=%s consumer=%s",
		notificationStream, consumerGroup, s.consumerName)
	return nil
}

// ensureGroup creates the consumer group on the stream. MKSTREAM lets us
// boot before any producer has run; BUSYGROUP is ignored so restarts are
// idempotent.
func (s *subscriber) ensureGroup(ctx context.Context) error {
	client := s.redis.GetClient()
	err := client.XGroupCreateMkStream(ctx, notificationStream, consumerGroup, "$").Err()
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return err
}

func (s *subscriber) listen(ctx context.Context) {
	defer s.wg.Done()
	client := s.redis.GetClient()

	for {
		select {
		case <-s.quit:
			return
		default:
		}

		// 5s block lets the goroutine wake periodically so Shutdown can fire
		// without waiting for the next produced entry.
		streams, err := client.XReadGroup(ctx, &goredis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: s.consumerName,
			Streams:  []string{notificationStream, ">"},
			Count:    streamReadCount,
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, goredis.Nil) {
				continue
			}
			select {
			case <-s.quit:
				return
			default:
			}
			// Redis runs on emptyDir, so a pod restart wipes the stream and
			// its consumer group; the next XReadGroup then loops forever with
			// NOGROUP. Re-run ensureGroup whenever Redis says the group is
			// gone so the subscriber heals on its own instead of spamming
			// errors until a deploy restarts it.
			if strings.Contains(err.Error(), "NOGROUP") {
				if recreated := s.ensureGroup(ctx); recreated != nil {
					s.logger.Errorf(ctx, "notification-srv: ensureGroup retry failed: %v", recreated)
				} else {
					s.logger.Warnf(ctx, "notification-srv: stream lost; consumer group recreated")
				}
				time.Sleep(time.Second)
				continue
			}
			s.logger.Errorf(ctx, "notification-srv: XReadGroup failed: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				s.handleStreamMessage(ctx, msg)
			}
		}
	}
}

// handleStreamMessage decodes one stream entry and acks it after the
// websocket usecase has processed it. Ack runs whether the handler succeeded
// or not; failures still need to be cleared from the pending entries list to
// avoid replay storms on the next reconnect.
func (s *subscriber) handleStreamMessage(ctx context.Context, msg goredis.XMessage) {
	channel, _ := msg.Values["channel"].(string)
	payload, ok := extractPayload(msg.Values["payload"])
	if !ok {
		s.logger.Errorf(ctx, "notification-srv: dropping malformed stream entry id=%s", msg.ID)
		s.ack(ctx, msg.ID)
		return
	}

	s.dispatch(ctx, channel, payload)
	s.ack(ctx, msg.ID)
}

func (s *subscriber) ack(ctx context.Context, id string) {
	if err := s.redis.GetClient().XAck(ctx, notificationStream, consumerGroup, id).Err(); err != nil {
		s.logger.Warnf(ctx, "notification-srv: XAck failed id=%s err=%v", id, err)
	}
}

func extractPayload(raw interface{}) ([]byte, bool) {
	switch v := raw.(type) {
	case string:
		return []byte(v), true
	case []byte:
		return v, true
	default:
		return nil, false
	}
}

func (s *subscriber) Shutdown(ctx context.Context) error {
	close(s.quit)
	s.wg.Wait()
	s.logger.Infof(ctx, "Redis stream consumer stopped")
	return nil
}
