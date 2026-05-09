package analyticsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
	sharedredis "github.com/smap-hcmut/shared-libs/go/redis"
)

const (
	defaultGroupID      = "notification-service"
	defaultDigestTopic  = "analytics.report.digest"
	defaultCrisisTopic  = "analytics.crisis.alert"
	defaultRedisChannel = "system:analytics"
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
		logger:       b.logger,
		redisClient:  b.redisClient,
		redisChannel: b.redisChannel,
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
	logger       log.Logger
	redisClient  sharedredis.IRedis
	redisChannel string
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
			h.logger.Errorf(session.Context(), "Failed to bridge analytics digest to websocket: topic=%s partition=%d offset=%d err=%v", msg.Topic, msg.Partition, msg.Offset, err)
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

	if err := h.redisClient.GetClient().Publish(ctx, h.redisChannel, payloadBytes).Err(); err != nil {
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
	ok, err := h.redisClient.GetClient().SetNX(ctx, dedupeKey, "1", dedupeTTL).Result()
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
	if err := h.redisClient.GetClient().Publish(ctx, channel, payloadBytes).Err(); err != nil {
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
