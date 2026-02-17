# Notification Service Adaptation Plan

**Service hiện tại:** `websocket-srv`  
**Service mới:** `notification-service`  
**Mục đích:** Adapt từ SaaS crawling platform → On-Premise Enterprise Analytics Platform

---

## 1. TỔNG QUAN THAY ĐỔI

### 1.1 Context Shift

| Aspect                 | Cũ (SaaS Crawling)                  | Mới (On-Premise Analytics)                        |
| ---------------------- | ----------------------------------- | ------------------------------------------------- |
| **Use Case**           | Crawl progress tracking             | Analytics pipeline + Crisis alerts                |
| **Entities**           | Project, Job, Batch                 | Project, Campaign, Data Source, Analytics         |
| **Notification Types** | Crawl status, Job progress          | Data onboarding, Analysis progress, Crisis alerts |
| **Channels**           | `user_noti:*`, `project:*`, `job:*` | `project:*`, `campaign:*`, `alert:*`, `system:*`  |
| **Deployment**         | Multi-tenant SaaS                   | Single-tenant On-Premise                          |

### 1.2 Vai trò mới

```
┌─────────────────────────────────────────────────────────────────┐
│                    NOTIFICATION SERVICE                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  INPUT (Redis Pub/Sub):                                         │
│  ├── Data onboarding progress (Ingest Service)                  │
│  ├── Analytics pipeline progress (Analytics Service)            │
│  ├── Crisis alerts (Analytics Service)                          │
│  └── System events (All services)                               │
│                                                                 │
│  PROCESSING:                                                    │
│  ├── WebSocket push (in-app notifications)                      │
│  ├── Discord webhook (team alerts)                              │
│  └── Alert history tracking                                     │
│                                                                 │
│  OUTPUT:                                                        │
│  ├── Real-time UI updates                                       │
│  ├── Team collaboration alerts                                  │
│  └── Audit trail                                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. NHỮNG GÌ GIỮ NGUYÊN ✅

### 2.1 Core Architecture

- ✅ **WebSocket Hub pattern** - Proven, scalable
- ✅ **Redis Pub/Sub** - Message broker
- ✅ **Gorilla WebSocket** - Library
- ✅ **JWT Authentication** - Via HttpOnly cookie
- ✅ **Graceful Shutdown** - Connection cleanup
- ✅ **Rate Limiting** - Max connections per user
- ✅ **Ping/Pong** - Keep-alive mechanism

### 2.2 Code Structure

```
internal/
├── websocket/
│   ├── hub.go          ✅ Keep (minor updates)
│   ├── connection.go   ✅ Keep (minor updates)
│   └── handler.go      ✅ Keep (update filters)
├── auth/
│   ├── jwt.go          ✅ Keep
│   ├── authorizer.go   ✅ Keep (update logic)
│   └── rate_limiter.go ✅ Keep
├── redis/
│   └── subscriber.go   🔄 Refactor (new channels)
├── transform/
│   └── transformer.go  🔄 Refactor (new message types)
└── types/
    ├── input.go        🔄 Refactor (new structs)
    └── output.go       🔄 Refactor (new structs)
```

---

## 3. NHỮNG GÌ CẦN XÓA ❌

### 3.1 Legacy Entities

```go
// ❌ DELETE: Job-related types (không còn crawl jobs)
type JobInputMessage struct {
    Platform string
    Status   string
    Batch    BatchData
    Progress int
}

type JobNotificationMessage struct {
    Platform string
    Status   string
    Batch    BatchData
    Progress int
}

// ❌ DELETE: Batch content (không còn crawl content)
type BatchData struct {
    ContentList []ContentItem
    TotalCount  int
}

type ContentItem struct {
    ID        string
    Text      string
    Author    AuthorInfo
    Metrics   MetricsInfo
    Media     []MediaItem
    Permalink string
}
```

### 3.2 Legacy Channels

```go
// ❌ DELETE: Old channel patterns
const (
    ChannelPatternUserNoti = "user_noti:*"  // Legacy
    ChannelPatternJob      = "job:*"        // No more jobs
)
```

### 3.3 Legacy Transform Logic

```go
// ❌ DELETE: Job transformation
func TransformJobMessage(channel string, payload []byte) (*JobNotificationMessage, error) {
    // ... delete entire function
}

// ❌ DELETE: Batch deduplication
func deduplicateContentItems(items []ContentItem) []ContentItem {
    // ... delete entire function
}
```

---

## 4. NHỮNG GÌ CẦN THÊM MỚI ➕

### 4.1 New Message Types

#### A. Data Onboarding Progress

```go
// internal/types/input.go
type DataOnboardingInputMessage struct {
    Type    string                 `json:"type"`    // "DATA_ONBOARDING"
    Payload DataOnboardingPayload  `json:"payload"`
}

type DataOnboardingPayload struct {
    ProjectID   string  `json:"project_id"`
    SourceID    string  `json:"source_id"`
    SourceName  string  `json:"source_name"`
    SourceType  string  `json:"source_type"`  // FILE_UPLOAD, WEBHOOK, TIKTOK, YOUTUBE
    Status      string  `json:"status"`       // UPLOADING, MAPPING, TRANSFORMING, COMPLETED, FAILED
    Progress    int     `json:"progress"`     // 0-100
    RecordCount int     `json:"record_count"`
    ErrorCount  int     `json:"error_count"`
    Message     string  `json:"message"`
}

// internal/types/output.go
type DataOnboardingNotification struct {
    Type      string                 `json:"type"`
    Timestamp string                 `json:"timestamp"`
    Payload   DataOnboardingPayload  `json:"payload"`
}
```

**Use Case:**

```
User uploads Excel file
    ↓
Ingest Service: PUBLISH project:{project_id}:user:{user_id}
    {
      "type": "DATA_ONBOARDING",
      "payload": {
        "project_id": "proj_vf8",
        "source_id": "src_001",
        "source_name": "Feedback Q1.xlsx",
        "source_type": "FILE_UPLOAD",
        "status": "MAPPING",
        "progress": 30,
        "message": "AI Schema Agent đang phân tích..."
      }
    }
    ↓
Notification Service → WebSocket push
    ↓
UI shows progress bar: "Đang phân tích schema... 30%"
```

---

#### B. Analytics Pipeline Progress

```go
// internal/types/input.go
type AnalyticsPipelineInputMessage struct {
    Type    string                    `json:"type"`    // "ANALYTICS_PIPELINE"
    Payload AnalyticsPipelinePayload  `json:"payload"`
}

type AnalyticsPipelinePayload struct {
    ProjectID       string  `json:"project_id"`
    SourceID        string  `json:"source_id"`
    TotalRecords    int     `json:"total_records"`
    ProcessedCount  int     `json:"processed_count"`
    SuccessCount    int     `json:"success_count"`
    FailedCount     int     `json:"failed_count"`
    Progress        int     `json:"progress"`  // 0-100
    CurrentPhase    string  `json:"current_phase"`  // SENTIMENT, ASPECT, KEYWORD
    EstimatedTimeMs int64   `json:"estimated_time_ms"`
}

// internal/types/output.go
type AnalyticsPipelineNotification struct {
    Type      string                    `json:"type"`
    Timestamp string                    `json:"timestamp"`
    Payload   AnalyticsPipelinePayload  `json:"payload"`
}
```

**Use Case:**

```
Analytics Service processing 1000 UAP records
    ↓
Every 100 records: PUBLISH project:{project_id}:user:{user_id}
    {
      "type": "ANALYTICS_PIPELINE",
      "payload": {
        "project_id": "proj_vf8",
        "source_id": "src_001",
        "total_records": 1000,
        "processed_count": 300,
        "progress": 30,
        "current_phase": "SENTIMENT",
        "estimated_time_ms": 120000
      }
    }
    ↓
UI shows: "Đang phân tích sentiment... 300/1000 (30%) - Còn ~2 phút"
```

---

#### C. Crisis Alert

```go
// internal/types/input.go
type CrisisAlertInputMessage struct {
    Type    string             `json:"type"`    // "CRISIS_ALERT"
    Payload CrisisAlertPayload `json:"payload"`
}

type CrisisAlertPayload struct {
    ProjectID       string   `json:"project_id"`
    ProjectName     string   `json:"project_name"`
    Severity        string   `json:"severity"`  // LOW, MEDIUM, HIGH, CRITICAL
    AlertType       string   `json:"alert_type"` // NEGATIVE_SPIKE, VIRAL_NEGATIVE, ASPECT_CRISIS
    Metric          string   `json:"metric"`     // "Negative sentiment"
    CurrentValue    float64  `json:"current_value"`
    Threshold       float64  `json:"threshold"`
    AffectedAspects []string `json:"affected_aspects"`
    SampleMentions  []string `json:"sample_mentions"`  // Top 3 negative mentions
    TimeWindow      string   `json:"time_window"`      // "Last 24 hours"
    ActionRequired  string   `json:"action_required"`
}

// internal/types/output.go
type CrisisAlertNotification struct {
    Type      string             `json:"type"`
    Timestamp string             `json:"timestamp"`
    Payload   CrisisAlertPayload `json:"payload"`
}
```

**Use Case:**

```
Analytics Service detects: Negative sentiment > 70% in last 24h
    ↓
PUBLISH alert:crisis:user:{user_id}
    {
      "type": "CRISIS_ALERT",
      "payload": {
        "project_id": "proj_vf8",
        "project_name": "Monitor VF8",
        "severity": "HIGH",
        "alert_type": "NEGATIVE_SPIKE",
        "metric": "Negative sentiment",
        "current_value": 0.75,
        "threshold": 0.70,
        "affected_aspects": ["BATTERY", "PRICE"],
        "sample_mentions": [
          "Pin sụt nhanh quá",
          "Giá quá đắt",
          "Không đáng tiền"
        ],
        "time_window": "Last 24 hours",
        "action_required": "Review negative feedback về PIN và GIÁ"
      }
    }
    ↓
Notification Service:
    1. WebSocket push → UI shows red alert banner
    2. Discord webhook → Post to #smap-alerts channel
```

---

#### D. Campaign Event

```go
// internal/types/input.go
type CampaignEventInputMessage struct {
    Type    string               `json:"type"`    // "CAMPAIGN_EVENT"
    Payload CampaignEventPayload `json:"payload"`
}

type CampaignEventPayload struct {
    CampaignID   string `json:"campaign_id"`
    CampaignName string `json:"campaign_name"`
    EventType    string `json:"event_type"`  // REPORT_GENERATED, ARTIFACT_CREATED
    ResourceID   string `json:"resource_id"`
    ResourceName string `json:"resource_name"`
    ResourceURL  string `json:"resource_url"`
    Message      string `json:"message"`
}

// internal/types/output.go
type CampaignEventNotification struct {
    Type      string               `json:"type"`
    Timestamp string               `json:"timestamp"`
    Payload   CampaignEventPayload `json:"payload"`
}
```

**Use Case:**

```
Knowledge Service completes report generation
    ↓
PUBLISH campaign:{campaign_id}:user:{user_id}
    {
      "type": "CAMPAIGN_EVENT",
      "payload": {
        "campaign_id": "camp_001",
        "campaign_name": "So sánh Xe điện Q1/2026",
        "event_type": "REPORT_GENERATED",
        "resource_id": "report_789",
        "resource_name": "Báo cáo so sánh Q1.pdf",
        "resource_url": "https://minio.smap.local/reports/report_789.pdf",
        "message": "Báo cáo đã sẵn sàng để tải xuống"
      }
    }
    ↓
UI shows notification: "Báo cáo 'So sánh Q1' đã hoàn thành" [Download]
```

---

### 4.2 New Redis Channels

```go
// internal/redis/channels.go
const (
    // Project-level notifications
    ChannelPatternProject = "project:*"  // Keep, update format

    // Campaign-level notifications (NEW)
    ChannelPatternCampaign = "campaign:*"

    // Crisis alerts (NEW)
    ChannelPatternAlert = "alert:*"

    // System-wide notifications (NEW)
    ChannelPatternSystem = "system:*"
)

// Channel format examples:
// - project:{project_id}:user:{user_id}
// - campaign:{campaign_id}:user:{user_id}
// - alert:crisis:user:{user_id}
// - alert:warning:user:{user_id}
// - system:maintenance:all
```

---

### 4.3 Alert Dispatcher Module (NEW)

```go
// internal/alerts/dispatcher.go
package alerts

import (
    "fmt"
    "time"
)

type AlertDispatcher struct {
    discordClient *DiscordClient
    config        *AlertConfig
}

type AlertConfig struct {
    DiscordWebhookURL string
}

func NewAlertDispatcher(config *AlertConfig) *AlertDispatcher {
    return &AlertDispatcher{
        discordClient: NewDiscordClient(config.DiscordWebhookURL),
        config:        config,
    }
}

// Dispatch crisis alert to Discord
func (d *AlertDispatcher) DispatchCrisisAlert(alert *CrisisAlertPayload) error {
    // Send to Discord
    if err := d.discordClient.SendCrisisAlert(alert); err != nil {
        return fmt.Errorf("discord: %w", err)
    }

    return nil
}

// Dispatch data onboarding progress (optional, for important milestones)
func (d *AlertDispatcher) DispatchDataOnboardingProgress(payload *DataOnboardingPayload) error {
    // Only send to Discord for COMPLETED or FAILED status
    if payload.Status != "COMPLETED" && payload.Status != "FAILED" {
        return nil
    }

    return d.discordClient.SendDataOnboardingProgress(payload)
}

// Dispatch campaign event
func (d *AlertDispatcher) DispatchCampaignEvent(payload *CampaignEventPayload) error {
    return d.discordClient.SendCampaignEvent(payload)
}
```

---

### 4.4 Discord Integration

````go
// internal/alerts/discord.go
package alerts

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

type DiscordClient struct {
    webhookURL string
    client     *http.Client
}

func NewDiscordClient(webhookURL string) *DiscordClient {
    return &DiscordClient{
        webhookURL: webhookURL,
        client:     &http.Client{Timeout: 10 * time.Second},
    }
}

type DiscordMessage struct {
    Content string         `json:"content,omitempty"`
    Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
    Title       string              `json:"title"`
    Description string              `json:"description"`
    Color       int                 `json:"color"`
    Fields      []DiscordEmbedField `json:"fields"`
    Timestamp   string              `json:"timestamp,omitempty"`
}

type DiscordEmbedField struct {
    Name   string `json:"name"`
    Value  string `json:"value"`
    Inline bool   `json:"inline"`
}

// SendCrisisAlert sends crisis alert to Discord with rich embed
func (c *DiscordClient) SendCrisisAlert(alert *CrisisAlertPayload) error {
    // Color mapping for severity
    color := map[string]int{
        "LOW":      3066993,  // Green
        "MEDIUM":   16776960, // Yellow
        "HIGH":     16737095, // Orange
        "CRITICAL": 15158332, // Red
    }[alert.Severity]

    // Build embed fields
    fields := []DiscordEmbedField{
        {Name: "Project", Value: alert.ProjectName, Inline: true},
        {Name: "Severity", Value: alert.Severity, Inline: true},
        {Name: "Metric", Value: alert.Metric, Inline: true},
        {Name: "Current Value", Value: fmt.Sprintf("%.1f%%", alert.CurrentValue*100), Inline: true},
        {Name: "Threshold", Value: fmt.Sprintf("%.1f%%", alert.Threshold*100), Inline: true},
        {Name: "Time Window", Value: alert.TimeWindow, Inline: true},
    }

    // Add affected aspects
    if len(alert.AffectedAspects) > 0 {
        fields = append(fields, DiscordEmbedField{
            Name:   "Affected Aspects",
            Value:  strings.Join(alert.AffectedAspects, ", "),
            Inline: false,
        })
    }

    // Add sample mentions
    if len(alert.SampleMentions) > 0 {
        mentions := "```\n"
        for i, mention := range alert.SampleMentions {
            if i >= 3 { // Limit to 3 samples
                break
            }
            mentions += fmt.Sprintf("%d. %s\n", i+1, mention)
        }
        mentions += "```"

        fields = append(fields, DiscordEmbedField{
            Name:   "Sample Mentions",
            Value:  mentions,
            Inline: false,
        })
    }

    // Add action required
    fields = append(fields, DiscordEmbedField{
        Name:   "Action Required",
        Value:  alert.ActionRequired,
        Inline: false,
    })

    // Build Discord message
    msg := DiscordMessage{
        Content: fmt.Sprintf("🚨 **Crisis Alert: %s**", alert.ProjectName),
        Embeds: []DiscordEmbed{
            {
                Title:       fmt.Sprintf("%s Alert - %s", alert.Severity, alert.AlertType),
                Description: fmt.Sprintf("**%s** has exceeded the threshold", alert.Metric),
                Color:       color,
                Fields:      fields,
                Timestamp:   time.Now().Format(time.RFC3339),
            },
        },
    }

    return c.sendMessage(msg)
}

// SendDataOnboardingProgress sends progress updates to Discord
func (c *DiscordClient) SendDataOnboardingProgress(payload *DataOnboardingPayload) error {
    color := 3447003 // Blue
    if payload.Status == "FAILED" {
        color = 15158332 // Red
    } else if payload.Status == "COMPLETED" {
        color = 3066993 // Green
    }

    msg := DiscordMessage{
        Embeds: []DiscordEmbed{
            {
                Title:       "📊 Data Onboarding " + payload.Status,
                Description: fmt.Sprintf("**%s** - %s", payload.SourceName, payload.SourceType),
                Color:       color,
                Fields: []DiscordEmbedField{
                    {Name: "Progress", Value: fmt.Sprintf("%d%%", payload.Progress), Inline: true},
                    {Name: "Records", Value: fmt.Sprintf("%d", payload.RecordCount), Inline: true},
                    {Name: "Errors", Value: fmt.Sprintf("%d", payload.ErrorCount), Inline: true},
                    {Name: "Message", Value: payload.Message, Inline: false},
                },
                Timestamp: time.Now().Format(time.RFC3339),
            },
        },
    }

    return c.sendMessage(msg)
}

// SendCampaignEvent sends campaign events to Discord
func (c *DiscordClient) SendCampaignEvent(payload *CampaignEventPayload) error {
    msg := DiscordMessage{
        Embeds: []DiscordEmbed{
            {
                Title:       "📢 Campaign Event",
                Description: fmt.Sprintf("**%s**", payload.CampaignName),
                Color:       5763719, // Purple
                Fields: []DiscordEmbedField{
                    {Name: "Event Type", Value: payload.EventType, Inline: true},
                    {Name: "Resource", Value: payload.ResourceName, Inline: true},
                    {Name: "Message", Value: payload.Message, Inline: false},
                    {Name: "Download", Value: fmt.Sprintf("[Click here](%s)", payload.ResourceURL), Inline: false},
                },
                Timestamp: time.Now().Format(time.RFC3339),
            },
        },
    }

    return c.sendMessage(msg)
}

// sendMessage is a helper to send message to Discord webhook
func (c *DiscordClient) sendMessage(msg DiscordMessage) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("marshal: %w", err)
    }

    resp, err := c.client.Post(c.webhookURL, "application/json", bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("post: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("discord returned %d", resp.StatusCode)
    }

    return nil
}
````

---

## 5. CONFIGURATION CHANGES

### 5.1 Environment Variables

```bash
# OLD (SaaS Crawling)
SERVICE_NAME=websocket-srv
REDIS_CHANNELS=user_noti:*,project:*,job:*
JWT_SECRET=xxx
CORS_ORIGINS=https://smap.tantai.dev

# NEW (On-Premise Analytics)
SERVICE_NAME=notification-service
REDIS_CHANNELS=project:*,campaign:*,alert:*,system:*
JWT_SECRET=xxx
CORS_ORIGINS=https://smap.local,https://smap.customer.com

# NEW: Discord Webhook Config
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/xxx/yyy

# NEW: Alert Thresholds
ALERT_CRISIS_THRESHOLD=0.70  # 70% negative sentiment
ALERT_WARNING_THRESHOLD=0.50 # 50% negative sentiment
```

### 5.2 Redis Channel Patterns

```go
// config/channels.go
package config

const (
    // Project-level notifications (KEEP, update format)
    // Format: project:{project_id}:user:{user_id}
    ChannelPatternProject = "project:*"

    // Campaign-level notifications (NEW)
    // Format: campaign:{campaign_id}:user:{user_id}
    ChannelPatternCampaign = "campaign:*"

    // Crisis alerts (NEW)
    // Format: alert:crisis:user:{user_id}
    // Format: alert:warning:user:{user_id}
    ChannelPatternAlert = "alert:*"

    // System-wide notifications (NEW)
    // Format: system:maintenance:all
    // Format: system:update:all
    ChannelPatternSystem = "system:*"
)

// GetAllChannels returns all channel patterns to subscribe
func GetAllChannels() []string {
    return []string{
        ChannelPatternProject,
        ChannelPatternCampaign,
        ChannelPatternAlert,
        ChannelPatternSystem,
    }
}
```

### 5.3 Message Type Registry

```go
// config/message_types.go
package config

const (
    // Data onboarding messages
    MessageTypeDataOnboarding = "DATA_ONBOARDING"

    // Analytics pipeline messages
    MessageTypeAnalyticsPipeline = "ANALYTICS_PIPELINE"

    // Crisis alert messages
    MessageTypeCrisisAlert = "CRISIS_ALERT"

    // Campaign event messages
    MessageTypeCampaignEvent = "CAMPAIGN_EVENT"

    // System messages
    MessageTypeSystemMaintenance = "SYSTEM_MAINTENANCE"
    MessageTypeSystemUpdate = "SYSTEM_UPDATE"
)

// MessageTypeConfig defines routing rules for each message type
type MessageTypeConfig struct {
    Type              string
    RequiresWebSocket bool
    RequiresDiscord   bool
    Priority          int // 1=LOW, 2=MEDIUM, 3=HIGH, 4=CRITICAL
}

var MessageTypeConfigs = map[string]MessageTypeConfig{
    MessageTypeDataOnboarding: {
        Type:              MessageTypeDataOnboarding,
        RequiresWebSocket: true,
        RequiresDiscord:   false, // Only send COMPLETED/FAILED to Discord
        Priority:          1,
    },
    MessageTypeAnalyticsPipeline: {
        Type:              MessageTypeAnalyticsPipeline,
        RequiresWebSocket: true,
        RequiresDiscord:   false, // Progress updates only in-app
        Priority:          1,
    },
    MessageTypeCrisisAlert: {
        Type:              MessageTypeCrisisAlert,
        RequiresWebSocket: true,
        RequiresDiscord:   true, // Critical alerts go to Discord
        Priority:          4,
    },
    MessageTypeCampaignEvent: {
        Type:              MessageTypeCampaignEvent,
        RequiresWebSocket: true,
        RequiresDiscord:   true, // Report ready notifications
        Priority:          2,
    },
}
```

---

## 6. DATABASE SCHEMA (Optional)

Notification Service vẫn **stateless**, nhưng có thể cần lưu **Alert History** cho audit trail.

### 6.1 Alert History Table (Optional)

```sql
-- Schema: notification (NEW)
CREATE SCHEMA IF NOT EXISTS notification;

-- Alert history for audit trail
CREATE TABLE notification.alert_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Alert metadata
    alert_type VARCHAR(50) NOT NULL,  -- CRISIS_ALERT, WARNING_ALERT
    severity VARCHAR(20) NOT NULL,    -- LOW, MEDIUM, HIGH, CRITICAL

    -- Related entities
    project_id VARCHAR(100) NOT NULL,
    project_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(100) NOT NULL,

    -- Alert content
    metric VARCHAR(100) NOT NULL,
    current_value DECIMAL(5,4) NOT NULL,
    threshold DECIMAL(5,4) NOT NULL,
    affected_aspects TEXT[],
    sample_mentions TEXT[],

    -- Dispatch status
    websocket_sent BOOLEAN DEFAULT FALSE,
    discord_sent BOOLEAN DEFAULT FALSE,

    -- Timestamps
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for query performance
CREATE INDEX idx_alert_history_project ON notification.alert_history(project_id);
CREATE INDEX idx_alert_history_user ON notification.alert_history(user_id);
CREATE INDEX idx_alert_history_triggered ON notification.alert_history(triggered_at DESC);
CREATE INDEX idx_alert_history_severity ON notification.alert_history(severity);

-- Retention policy: Auto-delete alerts older than 90 days
CREATE OR REPLACE FUNCTION notification.cleanup_old_alerts()
RETURNS void AS $$
BEGIN
    DELETE FROM notification.alert_history
    WHERE triggered_at < NOW() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- Schedule cleanup (run daily via cron or pg_cron)
-- SELECT cron.schedule('cleanup-alerts', '0 2 * * *', 'SELECT notification.cleanup_old_alerts()');
```

### 6.2 Alert Acknowledgement API (Optional)

```go
// internal/api/alerts.go
package api

// POST /api/alerts/{alert_id}/acknowledge
func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
    alertID := c.Param("alert_id")
    userID := c.GetString("user_id") // From JWT

    err := h.alertRepo.Acknowledge(alertID, userID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"message": "Alert acknowledged"})
}

// POST /api/alerts/{alert_id}/resolve
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
    alertID := c.Param("alert_id")
    userID := c.GetString("user_id")

    err := h.alertRepo.Resolve(alertID, userID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"message": "Alert resolved"})
}

// GET /api/alerts?project_id=xxx&status=open
func (h *AlertHandler) ListAlerts(c *gin.Context) {
    projectID := c.Query("project_id")
    status := c.Query("status") // open, acknowledged, resolved

    alerts, err := h.alertRepo.List(projectID, status)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, alerts)
}
```

---

## 7. TESTING STRATEGY

### 7.1 Unit Tests

```go
// internal/transform/transformer_test.go
package transform

func TestTransformDataOnboardingMessage(t *testing.T) {
    input := `{
        "type": "DATA_ONBOARDING",
        "payload": {
            "project_id": "proj_test",
            "source_id": "src_001",
            "source_name": "Test.xlsx",
            "source_type": "FILE_UPLOAD",
            "status": "MAPPING",
            "progress": 50,
            "record_count": 100,
            "error_count": 0,
            "message": "AI Schema Agent đang phân tích..."
        }
    }`

    channel := "project:proj_test:user:user123"
    output, err := TransformMessage(channel, []byte(input))

    assert.NoError(t, err)
    assert.Contains(t, string(output), "DATA_ONBOARDING")
    assert.Contains(t, string(output), "MAPPING")
}

func TestTransformCrisisAlert(t *testing.T) {
    input := `{
        "type": "CRISIS_ALERT",
        "payload": {
            "project_id": "proj_vf8",
            "project_name": "Monitor VF8",
            "severity": "HIGH",
            "alert_type": "NEGATIVE_SPIKE",
            "metric": "Negative sentiment",
            "current_value": 0.75,
            "threshold": 0.70,
            "affected_aspects": ["BATTERY", "PRICE"],
            "sample_mentions": ["Pin sụt nhanh", "Giá quá đắt"],
            "time_window": "Last 24 hours",
            "action_required": "Review negative feedback"
        }
    }`

    channel := "alert:crisis:user:user123"
    output, err := TransformMessage(channel, []byte(input))

    assert.NoError(t, err)
    assert.Contains(t, string(output), "CRISIS_ALERT")
    assert.Contains(t, string(output), "HIGH")
}
```

### 7.2 Integration Tests

```go
// tests/integration/notification_test.go
package integration

func TestDataOnboardingFlow(t *testing.T) {
    // 1. Setup Redis client
    rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

    // 2. Setup WebSocket client
    ws := connectWebSocket(t, "ws://localhost:8080/ws?token=test_jwt")
    defer ws.Close()

    // 3. Publish message to Redis
    payload := DataOnboardingInputMessage{
        Type: "DATA_ONBOARDING",
        Payload: DataOnboardingPayload{
            ProjectID:  "proj_test",
            SourceID:   "src_001",
            Status:     "MAPPING",
            Progress:   50,
            Message:    "Test message",
        },
    }

    data, _ := json.Marshal(payload)
    rdb.Publish(ctx, "project:proj_test:user:user123", data)

    // 4. Verify WebSocket receives message
    ws.SetReadDeadline(time.Now().Add(5 * time.Second))
    _, msg, err := ws.ReadMessage()

    assert.NoError(t, err)
    assert.Contains(t, string(msg), "DATA_ONBOARDING")
    assert.Contains(t, string(msg), "MAPPING")
}

func TestCrisisAlertDispatch(t *testing.T) {
    // Mock Discord webhook
    discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        body, _ := io.ReadAll(r.Body)
        assert.Contains(t, string(body), "Crisis Alert")
        w.WriteHeader(204) // Discord returns 204 No Content
    }))
    defer discordServer.Close()

    // Setup notification service with mock endpoint
    config := &AlertConfig{
        DiscordWebhookURL: discordServer.URL,
    }

    dispatcher := NewAlertDispatcher(config)

    // Dispatch crisis alert
    alert := &CrisisAlertPayload{
        ProjectID:    "proj_vf8",
        ProjectName:  "Monitor VF8",
        Severity:     "HIGH",
        AlertType:    "NEGATIVE_SPIKE",
        Metric:       "Negative sentiment",
        CurrentValue: 0.75,
        Threshold:    0.70,
    }

    err := dispatcher.DispatchCrisisAlert(alert)
    assert.NoError(t, err)
}
```

### 7.3 Load Tests

```bash
# Test WebSocket connection capacity
# Target: 10,000 concurrent connections

# Install k6
brew install k6

# Run load test
k6 run tests/load/websocket_load.js

# Expected results:
# - 10,000 connections: OK
# - Message latency: < 100ms (p95)
# - CPU usage: < 70%
# - Memory usage: < 2GB
```

```javascript
// tests/load/websocket_load.js
import ws from "k6/ws";
import { check } from "k6";

export let options = {
  stages: [
    { duration: "1m", target: 1000 }, // Ramp up to 1k
    { duration: "2m", target: 5000 }, // Ramp up to 5k
    { duration: "2m", target: 10000 }, // Ramp up to 10k
    { duration: "5m", target: 10000 }, // Stay at 10k
    { duration: "1m", target: 0 }, // Ramp down
  ],
};

export default function () {
  const url = "ws://localhost:8080/ws?token=test_jwt";

  const res = ws.connect(url, function (socket) {
    socket.on("open", () => console.log("Connected"));

    socket.on("message", (data) => {
      check(data, {
        "message received": (d) => d.length > 0,
      });
    });

    socket.on("close", () => console.log("Disconnected"));

    // Keep connection alive for 5 minutes
    socket.setTimeout(() => {
      socket.close();
    }, 300000);
  });

  check(res, { "status is 101": (r) => r && r.status === 101 });
}
```

---

## 8. MIGRATION TIMELINE

### 8.1 Phase 1: Code Refactoring (Week 7, Day 1-3)

| Task                                            | Duration | Owner | Status |
| ----------------------------------------------- | -------- | ----- | ------ |
| Rename `websocket-srv` → `notification-service` | 2h       | Dev   | ⏳     |
| Delete legacy types (Job, Batch, Content)       | 2h       | Dev   | ⏳     |
| Delete legacy channels (`user_noti:*`, `job:*`) | 1h       | Dev   | ⏳     |
| Delete legacy transform logic                   | 2h       | Dev   | ⏳     |
| Update Redis subscriber to new channels         | 4h       | Dev   | ⏳     |
| Add new message types (4 types)                 | 1d       | Dev   | ⏳     |
| Add new transform logic                         | 1d       | Dev   | ⏳     |
| Unit tests for new transforms                   | 4h       | Dev   | ⏳     |

**Deliverable:** Refactored codebase, legacy code removed

---

### 8.2 Phase 2: Discord Integration (Week 7, Day 4-5)

| Task                                     | Duration | Owner | Status |
| ---------------------------------------- | -------- | ----- | ------ |
| Implement Alert Dispatcher module        | 4h       | Dev   | ⏳     |
| Discord webhook integration              | 4h       | Dev   | ⏳     |
| Alert history database schema (optional) | 2h       | Dev   | ⏳     |
| Alert acknowledgement API (optional)     | 4h       | Dev   | ⏳     |
| Integration tests for alert dispatch     | 4h       | Dev   | ⏳     |

**Deliverable:** Discord alert system working

---

### 8.3 Phase 3: Testing & Deployment (Week 8, Day 1-2)

| Task                           | Duration | Owner  | Status |
| ------------------------------ | -------- | ------ | ------ |
| Integration tests (end-to-end) | 1d       | Dev    | ⏳     |
| Load tests (10k connections)   | 4h       | Dev    | ⏳     |
| Update Docker Compose config   | 2h       | DevOps | ⏳     |
| Update Helm charts             | 4h       | DevOps | ⏳     |
| Deploy to staging environment  | 2h       | DevOps | ⏳     |
| Smoke tests on staging         | 2h       | QA     | ⏳     |
| Deploy to production           | 1h       | DevOps | ⏳     |

**Deliverable:** Notification Service deployed to production

---

### 8.4 Total Timeline Summary

```
Week 7 (Notification Service Adaptation):
├── Day 1-3: Code Refactoring (3 days)
├── Day 4-5: Discord Integration (2 days)
└── Total: 5 days

Week 8 (Testing & Deployment):
├── Day 1-2: Testing & Deployment (2 days)
└── Total: 2 days

GRAND TOTAL: 7 days (1.4 weeks)
```

---

## 9. DEPLOYMENT CHECKLIST

### 9.1 Pre-Deployment

- [ ] **Code Review:** All PRs reviewed and approved
- [ ] **Unit Tests:** 100% pass rate
- [ ] **Integration Tests:** All scenarios pass
- [ ] **Load Tests:** 10k connections, < 100ms latency (p95)
- [ ] **Security Audit:** JWT validation, CORS config, rate limiting
- [ ] **Documentation:** API docs, runbook, troubleshooting guide

### 9.2 Configuration

- [ ] **Environment Variables:** All new env vars set in `.env`
- [ ] **Redis Channels:** Verify channel patterns in config
- [ ] **Discord Webhook:** Test webhook URL works
- [ ] **Alert Thresholds:** Confirm crisis/warning thresholds
- [ ] **CORS Origins:** Update for customer domain

### 9.3 Database (Optional)

- [ ] **Schema Migration:** Run `notification` schema creation
- [ ] **Indexes:** Verify indexes created
- [ ] **Retention Policy:** Schedule cleanup cron job
- [ ] **Backup:** Verify backup includes new schema

### 9.4 Deployment

- [ ] **Docker Image:** Build and push to registry
- [ ] **Helm Chart:** Update values.yaml with new config
- [ ] **Kubernetes:** Apply deployment manifest
- [ ] **Health Check:** Verify `/health` endpoint returns 200
- [ ] **Redis Connection:** Verify subscriber connected
- [ ] **WebSocket:** Test connection from browser

### 9.5 Post-Deployment Verification

- [ ] **Smoke Tests:**
  - [ ] Upload file → Receive DATA_ONBOARDING notification
  - [ ] Analytics complete → Receive ANALYTICS_PIPELINE notification
  - [ ] Trigger crisis → Receive CRISIS_ALERT (WebSocket + Discord)
  - [ ] Generate report → Receive CAMPAIGN_EVENT notification

- [ ] **Monitoring:**
  - [ ] Prometheus metrics scraping
  - [ ] Grafana dashboard showing connections, messages/sec
  - [ ] Alert rules configured (high error rate, connection drops)

- [ ] **Logging:**
  - [ ] Logs flowing to centralized logging (ELK/Loki)
  - [ ] Error logs monitored
  - [ ] Audit trail for alerts

### 9.6 Rollback Plan

If deployment fails:

1. **Immediate:** Revert Kubernetes deployment to previous version

   ```bash
   kubectl rollout undo deployment/notification-service
   ```

2. **Database:** If schema migration applied, rollback:

   ```sql
   DROP SCHEMA notification CASCADE;
   ```

3. **Redis:** No rollback needed (stateless)

4. **Notify Team:** Post incident report in Discord

---

## 10. SUCCESS METRICS

### 10.1 Performance Metrics

| Metric                    | Target            | Measurement                                   |
| ------------------------- | ----------------- | --------------------------------------------- |
| **WebSocket Connections** | 10,000 concurrent | Prometheus: `websocket_connections_total`     |
| **Message Latency**       | < 100ms (p95)     | Prometheus: `notification_message_latency_ms` |
| **Alert Dispatch Time**   | < 5 seconds       | Time from trigger to delivery                 |
| **Discord Delivery Rate** | > 99%             | Prometheus: `alert_discord_success_rate`      |
| **CPU Usage**             | < 70%             | Kubernetes metrics                            |
| **Memory Usage**          | < 2GB             | Kubernetes metrics                            |

### 10.2 Business Metrics

| Metric                    | Target       | Measurement                             |
| ------------------------- | ------------ | --------------------------------------- |
| **Crisis Alert Accuracy** | > 90%        | User feedback: false positive rate      |
| **Alert Response Time**   | < 10 minutes | Time from alert to user acknowledgement |
| **User Satisfaction**     | > 4/5 stars  | Post-deployment survey                  |

### 10.3 Operational Metrics

| Metric                           | Target       | Measurement                             |
| -------------------------------- | ------------ | --------------------------------------- |
| **Uptime**                       | > 99.9%      | Uptime monitoring                       |
| **Error Rate**                   | < 0.1%       | Prometheus: `notification_errors_total` |
| **Deployment Frequency**         | Weekly       | CI/CD pipeline                          |
| **Mean Time to Recovery (MTTR)** | < 30 minutes | Incident tracking                       |

---

## 11. APPENDIX

### 11.1 Message Flow Diagrams

```
DATA ONBOARDING FLOW:
─────────────────────
User uploads file
    ↓
Ingest Service: Transform to UAP
    ↓
PUBLISH project:{project_id}:user:{user_id}
    {
      "type": "DATA_ONBOARDING",
      "payload": {
        "status": "MAPPING",
        "progress": 30,
        ...
      }
    }
    ↓
Notification Service: Transform + Route
    ↓
WebSocket push to user's browser
    ↓
UI updates progress bar: "Đang phân tích schema... 30%"


CRISIS ALERT FLOW:
──────────────────
Analytics Service detects: Negative > 70%
    ↓
PUBLISH alert:crisis:user:{user_id}
    {
      "type": "CRISIS_ALERT",
      "payload": {
        "severity": "HIGH",
        "metric": "Negative sentiment",
        "current_value": 0.75,
        ...
      }
    }
    ↓
Notification Service: Transform + Dispatch
    ├─► WebSocket push → UI shows red banner
    └─► Discord webhook → Post to #smap-alerts channel
```

### 11.2 Discord Webhook Setup Guide

**1. Create Discord Server (if not exists):**

- Open Discord
- Click "+" → "Create My Own" → "For a club or community"
- Name: "SMAP Alerts"

**2. Create Webhook:**

- Go to Server Settings → Integrations → Webhooks
- Click "New Webhook"
- Name: "SMAP Notification Bot"
- Select channel: #smap-alerts
- Copy Webhook URL

**3. Test Webhook:**

```bash
curl -X POST "https://discord.com/api/webhooks/xxx/yyy" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "🚨 Test Alert",
    "embeds": [{
      "title": "Crisis Alert Test",
      "description": "This is a test message",
      "color": 15158332,
      "fields": [
        {"name": "Project", "value": "Test Project", "inline": true},
        {"name": "Severity", "value": "HIGH", "inline": true}
      ]
    }]
  }'
```

**4. Configure SMAP:**

```bash
# .env
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/xxx/yyy
```

### 11.3 Troubleshooting Guide

**Problem:** WebSocket connections drop frequently

**Solution:**

1. Check Redis connection: `redis-cli PING`
2. Check Nginx timeout: `proxy_read_timeout 3600s;`
3. Check client ping/pong: Browser console logs
4. Increase connection limit: `MAX_CONNECTIONS=20000`

---

**Problem:** Discord alerts not sending

**Solution:**

1. Verify webhook URL: Test with curl command above
2. Check network firewall: Allow outbound HTTPS to `discord.com`
3. Check logs: `kubectl logs -f deployment/notification-service | grep discord`
4. Verify payload format: Discord requires `embeds` array
5. Check rate limits: Discord allows 30 requests/minute per webhook

---

**Problem:** High memory usage

**Solution:**

1. Check connection count: `prometheus query: websocket_connections_total`
2. Check goroutine leaks: `curl localhost:6060/debug/pprof/goroutine`
3. Reduce buffer size: `SEND_BUFFER_SIZE=128` (default 256)
4. Enable connection limits: `MAX_CONNECTIONS_PER_USER=5`

---

### 11.4 References

- [Gorilla WebSocket Documentation](https://pkg.go.dev/github.com/gorilla/websocket)
- [Redis Pub/Sub Documentation](https://redis.io/docs/manual/pubsub/)
- [Discord Webhook Documentation](https://discord.com/developers/docs/resources/webhook)
- [Discord Embed Limits](https://discord.com/developers/docs/resources/channel#embed-limits)
- [SMAP Migration Plan v2.0](../planning/migration-plan-v2.md)
- [SMAP Analytics Schema](../knowledge/ANALYTICS_SCHEMA_EXPLAINED.md)
- [SMAP RAG Data Flow](../knowledge/RAG_DATA_FLOW_DETAILED.md)

---

**Document Version:** v1.0  
**Last Updated:** 16/02/2026  
**Author:** Nguyễn Tấn Tài  
**Status:** ✅ COMPLETE
