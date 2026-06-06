package analyticsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	goredis "github.com/redis/go-redis/v9"
	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
	sharedredis "github.com/smap-hcmut/shared-libs/go/redis"
)

const (
	defaultGroupID      = "notification-service"
	defaultDigestTopic  = "analytics.report.digest"
	defaultCrisisTopic  = "analytics.crisis.alert"
	defaultRedisChannel = "system:analytics"
	redisLogCooldown    = 60 * time.Second
	redisRetryAttempts  = 3
	redisRetryDelay     = 120 * time.Millisecond

	// notificationStream is the single Redis Stream that replaces the legacy
	// PubSub fan-out. PSubscribe had no backpressure — a slow WebSocket
	// subscriber silently dropped messages. The Stream keeps each entry until
	// every consumer group acks it, so notifications survive consumer lag.
	// The legacy channel name is preserved as a `channel` field on every
	// entry so the websocket router keeps the same routing semantics.
	notificationStream = "smap:notifications:stream"

	// notificationStreamMaxLen approximates the cap (~ minutes of traffic)
	// using XADD MAXLEN ~ so old entries are trimmed automatically. Set
	// generously because Redis cache is emptyDir per memory; longer history
	// is wasted if Redis restarts.
	notificationStreamMaxLen int64 = 10000
)

type Config struct {
	Enabled      bool
	Brokers      string
	GroupID      string
	DigestTopic  string
	CrisisTopic  string
	RedisChannel string
}

type Bridge struct {
	logger       log.Logger
	consumer     kafka.IConsumer
	redisClient  sharedredis.IRedis
	topics       []string
	redisChannel string
}

type digestPayload struct {
	ProjectID     string `json:"project_id"`
	CampaignID    string `json:"campaign_id"`
	RunID         string `json:"run_id"`
	Platform      string `json:"platform"`
	TotalMentions int    `json:"total_mentions"`
	MentionCount  int    `json:"mention_count"`
	DomainOverlay string `json:"domain_overlay"`
}

type crisisAlertPayload struct {
	AlertType       string   `json:"alert_type"`
	ProjectID       string   `json:"project_id"`
	ProjectName     string   `json:"project_name"`
	CampaignID      string   `json:"campaign_id,omitempty"`
	UserID          string   `json:"user_id"`
	Severity        string   `json:"severity"`
	Level           string   `json:"level,omitempty"`
	Metric          string   `json:"metric"`
	CurrentValue    float64  `json:"current_value"`
	Threshold       float64  `json:"threshold"`
	AffectedAspects []string `json:"affected_aspects"`
	SampleMentions  []string `json:"sample_mentions"`
	TimeWindow      string   `json:"time_window"`
	ActionRequired  string   `json:"action_required"`
	RunID           string   `json:"run_id,omitempty"`
	Title           string   `json:"title,omitempty"`
	Message         string   `json:"message,omitempty"`
	RepeatCooldown  int      `json:"repeat_cooldown_minutes,omitempty"`
	OpsAlert        bool     `json:"ops_alert,omitempty"`
	CreatedAt       string   `json:"created_at,omitempty"`
}

type analyticsPipelinePayload struct {
	ProjectID      string  `json:"project_id"`
	SourceID       string  `json:"source_id"`
	TotalRecords   int     `json:"total_records"`
	ProcessedCount int     `json:"processed_count"`
	SuccessCount   int     `json:"success_count"`
	FailedCount    int     `json:"failed_count"`
	Progress       float64 `json:"progress"`
	CurrentPhase   string  `json:"current_phase"`
	Message        string  `json:"message,omitempty"`
	CampaignID     string  `json:"campaign_id,omitempty"`
	RunID          string  `json:"run_id,omitempty"`
	Platform       string  `json:"platform,omitempty"`
	DomainOverlay  string  `json:"domain_overlay,omitempty"`
}

func ConfigFromEnv() Config {
	return Config{
		Enabled:      envBool("NOTIFICATION_KAFKA_ENABLED", false),
		Brokers:      firstNonEmpty(os.Getenv("NOTIFICATION_KAFKA_BROKERS"), os.Getenv("KAFKA_BROKERS"), "localhost:9092"),
		GroupID:      firstNonEmpty(os.Getenv("NOTIFICATION_KAFKA_GROUP_ID"), os.Getenv("KAFKA_GROUP_ID"), defaultGroupID),
		DigestTopic:  firstNonEmpty(os.Getenv("NOTIFICATION_DIGEST_TOPIC"), os.Getenv("ANALYTICS_DIGEST_TOPIC"), defaultDigestTopic),
		CrisisTopic:  firstNonEmpty(os.Getenv("NOTIFICATION_CRISIS_TOPIC"), os.Getenv("ANALYTICS_CRISIS_ALERT_TOPIC"), defaultCrisisTopic),
		RedisChannel: firstNonEmpty(os.Getenv("NOTIFICATION_REDIS_ANALYTICS_CHANNEL"), defaultRedisChannel),
	}
}

func NewFromEnv(ctx context.Context, logger log.Logger, redisClient sharedredis.IRedis) (*Bridge, error) {
	cfg := ConfigFromEnv()
	if !cfg.Enabled {
		logger.Infof(ctx, "Analytics notification bridge disabled")
		return nil, nil
	}
	return New(logger, redisClient, cfg)
}

func New(logger log.Logger, redisClient sharedredis.IRedis, cfg Config) (*Bridge, error) {
	brokers := splitCSV(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("analytics notification bridge requires at least one Kafka broker")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("analytics notification bridge requires Redis client")
	}

	groupID := firstNonEmpty(cfg.GroupID, defaultGroupID)
	digestTopic := firstNonEmpty(cfg.DigestTopic, defaultDigestTopic)
	crisisTopic := firstNonEmpty(cfg.CrisisTopic, defaultCrisisTopic)
	redisChannel := firstNonEmpty(cfg.RedisChannel, defaultRedisChannel)

	consumer, err := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: brokers,
		GroupID: groupID,
	})
	if err != nil {
		return nil, err
	}

	return &Bridge{
		logger:       logger,
		consumer:     consumer,
		redisClient:  redisClient,
		topics:       uniqueNonEmpty(digestTopic, crisisTopic),
		redisChannel: redisChannel,
	}, nil
}

func (b *Bridge) Start(ctx context.Context) {
	handler := &digestHandler{
		logger:          b.logger,
		redisClient:     b.redisClient,
		redisChannel:    b.redisChannel,
		publishErrByKey: map[string]time.Time{},
	}

	go b.drainErrors(ctx)
	go func() {
		b.logger.Infof(ctx, "Starting analytics notification bridge topics=%s channel=%s", strings.Join(b.topics, ","), b.redisChannel)
		for ctx.Err() == nil {
			if err := b.consumer.ConsumeWithContext(ctx, b.topics, handler); err != nil && ctx.Err() == nil {
				b.logger.Errorf(ctx, "Analytics notification bridge consume error: %v", err)
				time.Sleep(2 * time.Second)
			}
		}
	}()
}

func (b *Bridge) Close() error {
	return b.consumer.Close()
}

func (b *Bridge) drainErrors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-b.consumer.Errors():
			if !ok {
				return
			}
			if err != nil {
				b.logger.Errorf(ctx, "Analytics notification bridge consumer error: %v", err)
			}
		}
	}
}

type digestHandler struct {
	logger          log.Logger
	redisClient     sharedredis.IRedis
	redisChannel    string
	publishErrByKey map[string]time.Time
	mu              sync.Mutex
}

func (h *digestHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *digestHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *digestHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handleMessage(session.Context(), msg); err != nil {
			if isTransientRedisError(err) {
				h.logger.Warnf(
					session.Context(),
					"Retryable bridge failure: topic=%s partition=%d offset=%d err=%v",
					msg.Topic,
					msg.Partition,
					msg.Offset,
					err,
				)
				continue
			}
			h.logger.Errorf(
				session.Context(),
				"Failed to bridge analytics digest to websocket: topic=%s partition=%d offset=%d err=%v",
				msg.Topic,
				msg.Partition,
				msg.Offset,
				err,
			)
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

func (h *digestHandler) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	if strings.TrimSpace(msg.Topic) == defaultCrisisTopic || strings.Contains(strings.TrimSpace(msg.Topic), "crisis") {
		return h.handleCrisisAlert(ctx, msg)
	}

	var digest digestPayload
	if err := json.Unmarshal(msg.Value, &digest); err != nil {
		return fmt.Errorf("invalid digest payload: %w", err)
	}

	totalRecords := digest.TotalMentions
	if totalRecords == 0 {
		totalRecords = digest.MentionCount
	}

	payload := analyticsPipelinePayload{
		ProjectID:      digest.ProjectID,
		SourceID:       digest.Platform,
		TotalRecords:   totalRecords,
		ProcessedCount: totalRecords,
		SuccessCount:   totalRecords,
		FailedCount:    0,
		Progress:       100,
		CurrentPhase:   "report_digest_ready",
		Message:        "Analytics digest is ready",
		CampaignID:     digest.CampaignID,
		RunID:          digest.RunID,
		Platform:       digest.Platform,
		DomainOverlay:  digest.DomainOverlay,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if err := h.publishWithRetry(ctx, h.redisChannel, payloadBytes, "PUBLISH", h.redisChannel); err != nil {
		return err
	}

	h.logger.Infof(ctx, "Bridged analytics digest to websocket: project=%s campaign=%s platform=%s total_records=%d", digest.ProjectID, digest.CampaignID, digest.Platform, totalRecords)
	return nil
}

func (h *digestHandler) handleCrisisAlert(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var payload crisisAlertPayload
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		return fmt.Errorf("invalid crisis alert payload: %w", err)
	}
	payload.AlertType = firstNonEmpty(payload.AlertType, "CRISIS_ALERT")
	payload.Severity = strings.ToLower(firstNonEmpty(payload.Severity, "warning"))
	if strings.TrimSpace(payload.UserID) == "" {
		return fmt.Errorf("crisis alert missing user_id")
	}

	dedupeTTL := time.Duration(payload.RepeatCooldown) * time.Minute
	if dedupeTTL <= 0 {
		dedupeTTL = time.Hour
	}

	dedupeKey := fmt.Sprintf("notification:crisis:%s:%s:%s", payload.ProjectID, strings.ToUpper(payload.Level), payload.UserID)
	ok, err := h.setNXWithRetry(ctx, dedupeKey, dedupeTTL, payload.UserID)
	if err != nil {
		return err
	}
	if !ok {
		h.logger.Infof(ctx, "Skipped duplicate crisis alert: project=%s user=%s level=%s", payload.ProjectID, payload.UserID, payload.Level)
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	channel := fmt.Sprintf("alert:crisis:user:%s", payload.UserID)
	if err := h.publishWithRetry(ctx, channel, payloadBytes, "PUBLISH", payload.UserID); err != nil {
		return err
	}

	h.logger.Infof(ctx, "Bridged crisis alert to websocket: project=%s user=%s severity=%s", payload.ProjectID, payload.UserID, payload.Severity)
	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func uniqueNonEmpty(values ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// publishWithRetry pushes a notification into the shared Redis Stream. The
// channel argument is preserved as a payload field so consumers can route
// the same way they did under PubSub; switching to a stream gives the
// notification a durable queue position that survives slow subscribers.
func (h *digestHandler) publishWithRetry(ctx context.Context, channel string, payload []byte, operation, key string) error {
	var lastErr error
	args := &goredis.XAddArgs{
		Stream: notificationStream,
		MaxLen: notificationStreamMaxLen,
		Approx: true,
		Values: map[string]interface{}{
			"channel": channel,
			"payload": payload,
		},
	}
	for attempt := 1; attempt <= redisRetryAttempts; attempt++ {
		err := h.redisClient.GetClient().XAdd(ctx, args).Err()
		if err == nil {
			return nil
		}

		lastErr = err
		retryable := isTransientRedisError(err)
		if retryable && attempt < redisRetryAttempts {
			h.logRedisWarn(ctx, operation, key, attempt, redisRetryAttempts, err)
			time.Sleep(redisRetryDelay)
			continue
		}

		h.logRedisError(ctx, operation, key, err, retryable)
		if retryable {
			return lastErr
		}
		return lastErr
	}

	return lastErr
}

func (h *digestHandler) setNXWithRetry(ctx context.Context, key string, ttl time.Duration, keyLabel string) (bool, error) {
	var lastErr error
	for attempt := 1; attempt <= redisRetryAttempts; attempt++ {
		ok, err := h.redisClient.GetClient().SetNX(ctx, key, "1", ttl).Result()
		if err == nil {
			return ok, nil
		}

		lastErr = err
		retryable := isTransientRedisError(err)
		if retryable && attempt < redisRetryAttempts {
			h.logRedisWarn(ctx, "SETNX", keyLabel, attempt, redisRetryAttempts, err)
			time.Sleep(redisRetryDelay)
			continue
		}
		h.logRedisError(ctx, "SETNX", key, err, retryable)
		return false, lastErr
	}

	return false, lastErr
}

func (h *digestHandler) logRedisWarn(ctx context.Context, operation, key string, attempt, maxAttempts int, err error) {
	eventKey := fmt.Sprintf("%s:%s:transient", operation, key)
	now := time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()

	nextAt, exists := h.publishErrByKey[eventKey]
	if exists && now.Before(nextAt) {
		return
	}
	h.publishErrByKey[eventKey] = now.Add(redisLogCooldown)

	h.logger.Warnf(ctx, "%s transient redis error attempt=%d/%d key=%s: %v", operation, attempt, maxAttempts, key, err)
}

func (h *digestHandler) logRedisError(ctx context.Context, operation, key string, err error, retryable bool) {
	eventKey := fmt.Sprintf("%s:%s", operation, key)
	now := time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()

	nextAt, exists := h.publishErrByKey[eventKey]
	if exists && now.Before(nextAt) {
		return
	}
	h.publishErrByKey[eventKey] = now.Add(redisLogCooldown)

	if retryable {
		h.logger.Warnf(ctx, "Redis transient error: operation=%s key=%s err=%v", operation, key, err)
		return
	}
	h.logger.Errorf(ctx, "Redis operation failed: operation=%s key=%s err=%v", operation, key, err)
}

func isTransientRedisError(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	transientSignals := []string{
		"loading redis is loading the dataset in memory",
		"connect: connection refused",
		"connection refused",
		"connection timed out",
		"timeout",
		"read: connection reset",
		"connection reset",
		"temporary failure",
		"i/o timeout",
		"dial tcp",
	}
	for _, signal := range transientSignals {
		if strings.Contains(errText, signal) {
			return true
		}
	}
	return false
}
