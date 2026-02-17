package redis

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

type subscriberImpl struct {
	*subscriber // Embed the base struct to implement interface
	pubsub      *redis.PubSub
	wg          sync.WaitGroup
	quit        chan struct{}
}

// Ensure subscriber implements Subscriber interface.
// Since subscriber struct in new.go has fields but no methods on *subscriber,
// we might need to adjust how New returns.
// In new.go: func New(...) Subscriber { return &subscriber{...} }
// But subscriber struct doesn't have pubsub field.
// I should redefine subscriber struct in new.go or here to include pubsub.
// Or better, let's redefine the methods on *subscriber if I can add fields to it.
// I cannot easily add fields to a struct defined in another file if it's already defined.
// But `subscriber` struct IS defined in `new.go`.
// I should probably move the struct definition to `subscriber.go` or `new.go` and include all fields.
// Or just put everything in `subscriber.go`.
// Let's rewrite `new.go` to include `pubsub` field in `subscriber` struct,
// OR just define `subscriber` struct in `subscriber.go` (and remove from `new.go`?? No, `New` returns it).
// The cleanest way: redefine `subscriber` struct in `subscriber.go` and `New` in `new.go` uses it?
// Go doesn't allow duplicate struct definitions in same package.
// I will rewrite `new.go` to include the extra fields needed for lifecycle.

// Wait, I can just use a new struct `subscriberImpl` and return that in `New`.
// The `New` function in `new.go` currently returns `&subscriber{...}`.
// I will rewrite `new.go` to include the fields.
// But first let's see what `new.go` has. It has `subscriber` struct.
// I will overwrite `subscriber.go` with methods on `subscriber` struct,
// assuming I will ALSO update `new.go` to add the fields.

// Let's do `new.go` update first in this tool call sequence?
// I can do `write_to_file` for `new.go` then `subscriber.go`.
// `new.go` will have the full struct definition.
// `subscriber.go` will have the methods.

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
				s.logger.Warnf(ctx, "redis pubsub channel closed")
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
