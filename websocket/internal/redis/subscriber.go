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

	"smap-websocket/internal/transform"
	ws "smap-websocket/internal/websocket"
)

// Subscriber handles Redis Pub/Sub subscriptions
type Subscriber struct {
	client *redis.Client
	hub    *ws.Hub
	logger log.Logger

	// Transform layer
	transformer transform.MessageTransformer

	// Subscription management
	pubsub          *redis_client.PubSub
	subscriptions   map[string]bool // userID -> subscribed
	mu              sync.RWMutex
	patternChannels []string // Multiple patterns to subscribe to

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

	// Metrics
	messagesProcessed  int64
	transformErrors    int64
	lastTransformError string
}

// NewSubscriber creates a new Redis subscriber
func NewSubscriber(client *redis.Client, hub *ws.Hub, logger log.Logger) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())

	// Create transform layer dependencies
	validator := transform.NewInputValidator()
	metrics := transform.NewMetricsCollector()
	errorHandler := transform.NewErrorHandler(logger, metrics)
	transformer := transform.NewMessageTransformer(validator, metrics, errorHandler, logger)

	return &Subscriber{
		client:        client,
		hub:           hub,
		logger:        logger,
		transformer:   transformer,
		subscriptions: make(map[string]bool),
		patternChannels: []string{
			"user_noti:*", // Existing pattern for backward compatibility
			"project:*",   // New project notifications
			"job:*",       // New job notifications
		},
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
		maxRetries: 10,
		retryDelay: 5 * time.Second,
	}
}

// Start starts the Redis subscriber
func (s *Subscriber) Start() error {
	// Subscribe to multiple patterns
	s.pubsub = s.client.PSubscribe(s.ctx, s.patternChannels...)

	// Mark subscriber as active
	s.isActive.Store(true)

	s.logger.Infof(s.ctx, "Redis subscriber started, listening on patterns: %v", s.patternChannels)

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
	atomic.AddInt64(&s.messagesProcessed, 1)
	s.mu.Unlock()

	// Determine message type based on channel pattern
	channelParts := strings.Split(channel, ":")
	if len(channelParts) < 2 {
		s.logger.Warnf(s.ctx, "Invalid channel format: %s", channel)
		return
	}

	channelType := channelParts[0]

	switch channelType {
	case "user_noti":
		// Handle legacy user notification format
		s.handleLegacyUserNotification(channel, payload)

	case "project", "job":
		// Handle topic-based notifications with transform layer
		s.handleTopicNotification(channel, payload)

	default:
		s.logger.Warnf(s.ctx, "Unknown channel type: %s", channelType)
	}
}

// handleLegacyUserNotification handles existing user_noti:* messages for backward compatibility
func (s *Subscriber) handleLegacyUserNotification(channel string, payload string) {
	// Extract user ID from channel name: user_noti:{user_id}
	parts := strings.Split(channel, ":")
	if len(parts) != 2 {
		s.logger.Warnf(s.ctx, "Invalid legacy channel format: %s", channel)
		return
	}

	userID := parts[1]

	// Parse the message payload
	var redisMsg RedisMessage
	if err := json.Unmarshal([]byte(payload), &redisMsg); err != nil {
		s.logger.Errorf(s.ctx, "Failed to unmarshal legacy Redis message: %v", err)
		return
	}

	// Log dry-run messages specifically
	if redisMsg.IsDryRunResult() {
		var dryRunPayload map[string]any
		if err := json.Unmarshal(redisMsg.Payload, &dryRunPayload); err == nil {
			s.logger.Infof(s.ctx, "Received legacy dry-run result for user %s: job_id=%v, platform=%v, status=%v",
				userID, dryRunPayload["job_id"], dryRunPayload["platform"], dryRunPayload["status"])
		}
	}

	// Create WebSocket message
	wsMsg := &ws.Message{
		Type:      ws.MessageType(redisMsg.Type),
		Payload:   redisMsg.Payload,
		Timestamp: time.Now(),
	}

	// Send to Hub for delivery (legacy behavior - all connections)
	s.hub.SendToUser(userID, wsMsg)

	s.logger.Debugf(s.ctx, "Routed legacy message to user %s (type: %s)", userID, redisMsg.Type)
}

// handleTopicNotification handles project:* and job:* messages with transform layer
func (s *Subscriber) handleTopicNotification(channel string, payload string) {
	// Use transform layer to process and validate the message
	transformedMsg, err := s.transformer.TransformMessage(s.ctx, channel, payload)
	if err != nil {
		atomic.AddInt64(&s.transformErrors, 1)
		s.mu.Lock()
		s.lastTransformError = err.Error()
		s.mu.Unlock()
		s.logger.Errorf(s.ctx, "Transform failed for channel %s: %v", channel, err)
		return
	}

	// Extract userID and topic info from channel
	topicType, topicID, userID, err := s.parseTopicChannel(channel)
	if err != nil {
		s.logger.Errorf(s.ctx, "Failed to parse topic channel %s: %v", channel, err)
		return
	}

	// Create WebSocket message with transformed payload
	transformedBytes, err := json.Marshal(transformedMsg)
	if err != nil {
		s.logger.Errorf(s.ctx, "Failed to marshal transformed message: %v", err)
		return
	}

	wsMsg := &ws.Message{
		Type:      s.getWebSocketMessageType(topicType),
		Payload:   transformedBytes,
		Timestamp: time.Now(),
	}

	// Send to Hub with topic-based routing
	s.routeTopicMessage(topicType, topicID, userID, wsMsg)

	s.logger.Debugf(s.ctx, "Routed %s message to user %s for %s %s",
		topicType, userID, topicType, topicID)
}

// parseTopicChannel parses topic channel format and extracts components
func (s *Subscriber) parseTopicChannel(channel string) (topicType, topicID, userID string, err error) {
	return transform.ValidateTopicFormat(channel)
}

// getWebSocketMessageType maps topic type to WebSocket message type
func (s *Subscriber) getWebSocketMessageType(topicType string) ws.MessageType {
	switch topicType {
	case "project":
		return ws.MessageTypeProjectProgress
	case "job":
		return ws.MessageTypeJobProgress
	default:
		return ws.MessageTypeNotification
	}
}

// routeTopicMessage routes message based on topic type
func (s *Subscriber) routeTopicMessage(topicType, topicID, userID string, wsMsg *ws.Message) {
	switch topicType {
	case "project":
		// Send to project-specific connections using the implemented method
		s.hub.SendToUserWithProject(userID, topicID, wsMsg)
		s.logger.Debugf(s.ctx, "Sent project message to user %s for project %s", userID, topicID)

	case "job":
		// Send to job-specific connections using the implemented method
		s.hub.SendToUserWithJob(userID, topicID, wsMsg)
		s.logger.Debugf(s.ctx, "Sent job message to user %s for job %s", userID, topicID)

	default:
		// Fallback for unknown topic types
		s.hub.SendToUser(userID, wsMsg)
		s.logger.Debugf(s.ctx, "Sent general message to user %s", userID)
	}
}

// reconnect attempts to reconnect to Redis
func (s *Subscriber) reconnect() error {
	for i := 0; i < s.maxRetries; i++ {
		s.logger.Infof(s.ctx, "Reconnecting to Redis (attempt %d/%d)...", i+1, s.maxRetries)

		// Close old pubsub
		if s.pubsub != nil {
			s.pubsub.Close()
		}

		// Create new pubsub with all patterns
		s.pubsub = s.client.PSubscribe(s.ctx, s.patternChannels...)

		// Test the connection
		if _, err := s.pubsub.Receive(s.ctx); err == nil {
			s.logger.Infof(s.ctx, "Successfully reconnected to Redis with patterns: %v", s.patternChannels)
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

	// Join multiple patterns with comma for compatibility with server interface
	pattern = strings.Join(s.patternChannels, ",")
	return s.isActive.Load(), lastMsg, pattern
}

// GetSubscriberMetrics returns current subscriber metrics
func (s *Subscriber) GetSubscriberMetrics() SubscriberMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SubscriberMetrics{
		MessagesProcessed:  atomic.LoadInt64(&s.messagesProcessed),
		TransformErrors:    atomic.LoadInt64(&s.transformErrors),
		LastTransformError: s.lastTransformError,
		LastMessageAt:      s.lastMessageAt,
		IsActive:           s.isActive.Load(),
		Patterns:           s.patternChannels,
	}
}

// SubscriberMetrics represents subscriber performance metrics
type SubscriberMetrics struct {
	MessagesProcessed  int64     `json:"messages_processed"`
	TransformErrors    int64     `json:"transform_errors"`
	LastTransformError string    `json:"last_transform_error,omitempty"`
	LastMessageAt      time.Time `json:"last_message_at"`
	IsActive           bool      `json:"is_active"`
	Patterns           []string  `json:"patterns"`
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
