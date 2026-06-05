package redis

import (
	"context"
	"fmt"
)

func (s *subscriber) Start() error {
	ctx := context.Background()

	channels := []string{
		"project:*:user:*",
		"campaign:*:user:*",
		"alert:*:user:*",
		"system:*",
	}

	// Get underlying client
	client := s.redis.GetClient()
	s.pubsub = client.PSubscribe(ctx, channels...)

	// Wait for confirmation that subscription is created
	_, err := s.pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	s.wg.Add(1)
	go s.listen(ctx)

	s.logger.Infof(ctx, "Redis subscriber started on channels: %v", channels)
	return nil
}

func (s *subscriber) listen(ctx context.Context) {
	defer s.wg.Done()

	ch := s.pubsub.Channel()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				select {
				case <-s.quit:
					// Normal shutdown — pubsub closed as part of Shutdown()
				default:
					s.logger.Errorf(ctx, "notification-srv: redis pubsub channel closed unexpectedly — notifications halted")
				}
				return
			}
			s.handleMessage(ctx, msg)
		case <-s.quit:
			return
		}
	}
}

func (s *subscriber) Shutdown(ctx context.Context) error {
	close(s.quit)
	if s.pubsub != nil {
		if err := s.pubsub.Close(); err != nil {
			s.logger.Errorf(ctx, "failed to close pubsub: %v", err)
		}
	}
	s.wg.Wait()
	s.logger.Infof(ctx, "Redis subscriber stopped")
	return nil
}
