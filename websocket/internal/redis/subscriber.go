package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	redis_client "github.com/redis/go-redis/v9"

	"smap-websocket/pkg/log"
	"smap-websocket/pkg/redis"

	ws "smap-websocket/internal/websocket"
)

// Subscriber handles Redis Pub/Sub subscriptions
type Subscriber struct {
	client *redis.Client
	hub    *ws.Hub
	logger log.Logger

	// Subscription management
	pubsub         *redis_client.PubSub
	subscriptions  map[string]bool // userID -> subscribed
	mu             sync.RWMutex
	patternChannel string

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// Reconnection settings
	maxRetries int
	retryDelay time.Duration

	// Health tracking
	lastMessageAt time.Time
	isActive      atomic.Bool
}

// NewSubscriber creates a new Redis subscriber
func NewSubscriber(client *redis.Client, hub *ws.Hub, logger log.Logger) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())

	return &Subscriber{
		client:         client,
		hub:            hub,
		logger:         logger,
		subscriptions:  make(map[string]bool),
		patternChannel: "user_noti:*",
		ctx:            ctx,
		cancel:         cancel,
		done:           make(chan struct{}),
		maxRetries:     10,
		retryDelay:     5 * time.Second,
	}
}

// Start starts the Redis subscriber
func (s *Subscriber) Start() error {
	// Subscribe to the pattern
	s.pubsub = s.client.PSubscribe(s.ctx, s.patternChannel)

	// Mark subscriber as active
	s.isActive.Store(true)

	s.logger.Infof(s.ctx, "Redis subscriber started, listening on pattern: %s", s.patternChannel)

	// Start listening in a goroutine
	go s.listen()

	return nil
}

// listen listens for messages from Redis and routes them to the Hub
func (s *Subscriber) listen() {
	defer close(s.done)

	ch := s.pubsub.Channel()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info(context.Background(), "Redis subscriber shutting down...")
			return

		case msg, ok := <-ch:
			if !ok {
				s.logger.Error(s.ctx, "Redis pub/sub channel closed, attempting to reconnect...")
				if err := s.reconnect(); err != nil {
					s.logger.Errorf(s.ctx, "Failed to reconnect to Redis: %v", err)
					return
				}
				ch = s.pubsub.Channel()
				continue
			}

			// Handle the message
			s.handleMessage(msg.Channel, msg.Payload)
		}
	}
}

// handleMessage processes a message from Redis
func (s *Subscriber) handleMessage(channel string, payload string) {
	// Track last message timestamp
	s.mu.Lock()
	s.lastMessageAt = time.Now()
	s.mu.Unlock()

	// Extract user ID from channel name: user_noti:{user_id}
	parts := strings.Split(channel, ":")
	if len(parts) != 2 {
		s.logger.Warnf(s.ctx, "Invalid channel format: %s", channel)
		return
	}

	userID := parts[1]

	// Parse the message payload
	var redisMsg RedisMessage
	if err := json.Unmarshal([]byte(payload), &redisMsg); err != nil {
		s.logger.Errorf(s.ctx, "Failed to unmarshal Redis message: %v", err)
		return
	}

	// Log dry-run messages specifically
	if redisMsg.IsDryRunResult() {
		var dryRunPayload map[string]any
		if err := json.Unmarshal(redisMsg.Payload, &dryRunPayload); err == nil {
			s.logger.Infof(s.ctx, "Received dry-run result for user %s: job_id=%v, platform=%v, status=%v",
				userID, dryRunPayload["job_id"], dryRunPayload["platform"], dryRunPayload["status"])
		}
	}

	// Create WebSocket message
	wsMsg := &ws.Message{
		Type:      ws.MessageType(redisMsg.Type),
		Payload:   redisMsg.Payload,
		Timestamp: time.Now(),
	}

	// Send to Hub for delivery
	s.hub.SendToUser(userID, wsMsg)

	s.logger.Debugf(s.ctx, "Routed message to user %s (type: %s)", userID, redisMsg.Type)
}

// reconnect attempts to reconnect to Redis
func (s *Subscriber) reconnect() error {
	for i := 0; i < s.maxRetries; i++ {
		s.logger.Infof(s.ctx, "Reconnecting to Redis (attempt %d/%d)...", i+1, s.maxRetries)

		// Close old pubsub
		if s.pubsub != nil {
			s.pubsub.Close()
		}

		// Create new pubsub
		s.pubsub = s.client.PSubscribe(s.ctx, s.patternChannel)

		// Test the connection
		if _, err := s.pubsub.Receive(s.ctx); err == nil {
			s.logger.Info(s.ctx, "Successfully reconnected to Redis")
			return nil
		}

		// Wait before retry
		time.Sleep(s.retryDelay)
	}

	return fmt.Errorf("failed to reconnect to Redis after %d attempts", s.maxRetries)
}

// OnUserConnected is called when a user connects (implements RedisNotifier interface)
func (s *Subscriber) OnUserConnected(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Mark as subscribed (note: we use pattern subscription, so individual subscriptions aren't needed)
	s.subscriptions[userID] = true

	s.logger.Debugf(s.ctx, "User %s marked as connected in Redis subscriber", userID)
	return nil
}

// OnUserDisconnected is called when a user disconnects (implements RedisNotifier interface)
func (s *Subscriber) OnUserDisconnected(userID string, hasOtherConnections bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If user has no other connections, remove from subscription tracking
	if !hasOtherConnections {
		delete(s.subscriptions, userID)
		s.logger.Debugf(s.ctx, "User %s marked as disconnected in Redis subscriber", userID)
	}

	return nil
}

// GetHealthInfo returns the current health info of the subscriber
func (s *Subscriber) GetHealthInfo() (active bool, lastMessageAt time.Time, pattern string) {
	s.mu.RLock()
	lastMsg := s.lastMessageAt
	s.mu.RUnlock()

	return s.isActive.Load(), lastMsg, s.patternChannel
}

// Shutdown gracefully shuts down the subscriber
func (s *Subscriber) Shutdown(ctx context.Context) error {
	// Mark as inactive
	s.isActive.Store(false)

	s.cancel()

	// Close pubsub
	if s.pubsub != nil {
		if err := s.pubsub.Close(); err != nil {
			s.logger.Errorf(context.Background(), "Error closing pub/sub: %v", err)
		}
	}

	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Message type constants
const (
	MessageTypeDryRunResult = "dryrun_result"
)

// RedisMessage represents a message from Redis
type RedisMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// IsDryRunResult checks if the message is a dry-run result
func (r *RedisMessage) IsDryRunResult() bool {
	return r.Type == MessageTypeDryRunResult
}
