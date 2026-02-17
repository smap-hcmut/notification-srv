package websocket

import "time"

// --- Message Types ---
type MessageType string

const (
	MessageTypeDataOnboarding    MessageType = "DATA_ONBOARDING"
	MessageTypeAnalyticsPipeline MessageType = "ANALYTICS_PIPELINE"
	MessageTypeCrisisAlert       MessageType = "CRISIS_ALERT"
	MessageTypeCampaignEvent     MessageType = "CAMPAIGN_EVENT"
	MessageTypeSystem            MessageType = "SYSTEM"
)

// --- Channel Types ---
type ChannelType string

const (
	ChannelTypeProject  ChannelType = "project"
	ChannelTypeCampaign ChannelType = "campaign"
	ChannelTypeAlert    ChannelType = "alert"
	ChannelTypeSystem   ChannelType = "system"
)

// --- UseCase Inputs ---

// ProcessMessageInput is the raw input from Redis
type ProcessMessageInput struct {
	Channel string
	Payload []byte
}

// ConnectionInput represents a new connection attempt
type ConnectionInput struct {
	UserID    string
	ProjectID string      // Optional filter
	Conn      interface{} // *websocket.Conn (handled as interface{} to avoid direct dependency in public type if preferred, or wrapped)
}

// --- UseCase Outputs ---

type HubStats struct {
	ActiveConnections int
	TotalUniqueUsers  int
}

// NotificationOutput is the final payload sent to the client
type NotificationOutput struct {
	Type      MessageType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// --- Payload Types (for Transformation) ---

type DataOnboardingPayload struct {
	ProjectID   string `json:"project_id"`
	SourceID    string `json:"source_id"`
	SourceName  string `json:"source_name"`
	SourceType  string `json:"source_type"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	RecordCount int    `json:"record_count"`
	ErrorCount  int    `json:"error_count"`
	Message     string `json:"message"`
}

type AnalyticsPipelinePayload struct {
	ProjectID       string `json:"project_id"`
	SourceID        string `json:"source_id"`
	TotalRecords    int    `json:"total_records"`
	ProcessedCount  int    `json:"processed_count"`
	SuccessCount    int    `json:"success_count"`
	FailedCount     int    `json:"failed_count"`
	Progress        int    `json:"progress"`
	CurrentPhase    string `json:"current_phase"`
	EstimatedTimeMs int64  `json:"estimated_time_ms"`
}

type CrisisAlertPayload struct {
	ProjectID       string   `json:"project_id"`
	ProjectName     string   `json:"project_name"`
	Severity        string   `json:"severity"`
	AlertType       string   `json:"alert_type"`
	Metric          string   `json:"metric"`
	CurrentValue    float64  `json:"current_value"`
	Threshold       float64  `json:"threshold"`
	AffectedAspects []string `json:"affected_aspects"`
	SampleMentions  []string `json:"sample_mentions"`
	TimeWindow      string   `json:"time_window"`
	ActionRequired  string   `json:"action_required"`
}

type CampaignEventPayload struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	EventType    string `json:"event_type"`
	ResourceID   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	ResourceURL  string `json:"resource_url"`
	Message      string `json:"message"`
}
