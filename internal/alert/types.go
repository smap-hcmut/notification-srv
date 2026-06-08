package alert

import "time"

// CrisisSampleReference carries one piece of evidence behind a crisis
// alert: the source URL plus a short excerpt. analysis-srv looks each
// UAP up in analysis.post_insight before emitting the alert so the
// front-end toast and the Discord embed can link to the actual post.
type CrisisSampleReference struct {
	UapID          string
	UapType        string
	URL            string
	ContentExcerpt string
}

// CrisisAlertInput represents a critical system or business alert.
type CrisisAlertInput struct {
	ProjectID        string
	ProjectName      string
	Severity         string // e.g. "critical", "warning", "info"
	AlertType        string // e.g. "spike", "drop", "sentiment"
	Metric           string // e.g. "mention_count"
	CurrentValue     float64
	Threshold        float64
	AffectedAspects  []string
	SampleMentions   []string                // List of texts (legacy, kept for back-compat)
	SampleReferences []CrisisSampleReference // Source posts with URLs
	TimeWindow       string
	ActionRequired   string
	GeneratedAt      time.Time
}

// DataOnboardingInput represents a status update for a data source onboarding.
type DataOnboardingInput struct {
	ProjectID   string
	SourceID    string
	SourceName  string
	SourceType  string // e.g. "facebook_page", "instagram_account"
	Status      string // "completed", "failed"
	RecordCount int
	ErrorCount  int
	Message     string // Error details or success summary
	Duration    time.Duration
}

// CampaignEventInput represents a notification about a campaign state change.
type CampaignEventInput struct {
	CampaignID   string
	CampaignName string
	EventType    string // "created", "started", "paused", "finished"
	ResourceName string // e.g. "Keyword List", "Competitor List"
	ResourceURL  string
	User         string // Who triggered the event
	Message      string
	Timestamp    time.Time
}
