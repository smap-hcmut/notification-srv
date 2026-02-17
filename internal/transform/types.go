package transform

import (
	"encoding/json"
	"time"
)

// TransformMetrics defines metrics collected by the transform layer
type TransformMetrics struct {
	ProjectTransformSuccess int64
	ProjectTransformErrors  int64
	JobTransformSuccess     int64
	JobTransformErrors      int64
	ProjectTransformLatency time.Duration
	JobTransformLatency     time.Duration
	ValidationErrors        int64
	ValidationSuccess       int64
	JSONParseErrors         int64
	MissingFieldErrors      int64
	InvalidStatusErrors     int64
	InvalidPlatformErrors   int64
	InvalidValueErrors      int64
}

// ============================================================================
// Input Types (from internal/types/input.go)
// ============================================================================

// ProjectInputMessage represents the input structure from Publishers for project notifications
type ProjectInputMessage struct {
	Status   string         `json:"status"`             // "PROCESSING", "COMPLETED", "FAILED", "PAUSED"
	Progress *ProgressInput `json:"progress,omitempty"` // Overall progress
}

// JobInputMessage represents the input structure from Publishers for job notifications
type JobInputMessage struct {
	Platform string         `json:"platform"`           // "TIKTOK", "YOUTUBE", "INSTAGRAM"
	Status   string         `json:"status"`             // "PROCESSING", "COMPLETED", "FAILED", "PAUSED"
	Batch    *BatchInput    `json:"batch,omitempty"`    // Current batch data
	Progress *ProgressInput `json:"progress,omitempty"` // Overall job progress
}

// ProgressInput represents progress information from publishers
type ProgressInput struct {
	Current    int      `json:"current"`    // Current completed items
	Total      int      `json:"total"`      // Total items to process
	Percentage float64  `json:"percentage"` // Completion percentage (0-100)
	ETA        float64  `json:"eta"`        // Estimated time remaining in minutes
	Errors     []string `json:"errors"`     // Array of error messages
}

// BatchInput represents a batch of crawled content from publishers
type BatchInput struct {
	Keyword     string         `json:"keyword"`      // Search keyword
	ContentList []ContentInput `json:"content_list"` // Crawled content items
	CrawledAt   string         `json:"crawled_at"`   // ISO timestamp
}

// ContentInput represents a single social media content item from publishers
type ContentInput struct {
	ID          string       `json:"id"`              // Content unique ID
	Text        string       `json:"text"`            // Content text/caption
	Author      AuthorInput  `json:"author"`          // Author information
	Metrics     MetricsInput `json:"metrics"`         // Engagement statistics
	Media       *MediaInput  `json:"media,omitempty"` // Media information
	PublishedAt string       `json:"published_at"`    // ISO timestamp
	Permalink   string       `json:"permalink"`       // Direct link to content
}

// AuthorInput represents content author information from publishers
type AuthorInput struct {
	ID         string `json:"id"`          // Author unique ID
	Username   string `json:"username"`    // Author username/handle
	Name       string `json:"name"`        // Author display name
	Followers  int    `json:"followers"`   // Follower count
	IsVerified bool   `json:"is_verified"` // Verification status
	AvatarURL  string `json:"avatar_url"`  // Profile picture URL
}

// MetricsInput represents engagement metrics from publishers
type MetricsInput struct {
	Views    int     `json:"views"`    // View count
	Likes    int     `json:"likes"`    // Like count
	Comments int     `json:"comments"` // Comment count
	Shares   int     `json:"shares"`   // Share count
	Rate     float64 `json:"rate"`     // Engagement rate percentage
}

// MediaInput represents media information from publishers
type MediaInput struct {
	Type      string `json:"type"`               // "video", "image", "audio"
	Duration  int    `json:"duration,omitempty"` // Duration in seconds
	Thumbnail string `json:"thumbnail"`          // Thumbnail/preview URL
	URL       string `json:"url"`                // Media file URL
}

// Validate validates the project input message
func (p *ProjectInputMessage) Validate() error {
	// status is optional - Project status may not be in message format
	// Collector Service manages status in Redis, may not push to crawler
	// If status is provided, validate it; otherwise allow empty
	if p.Status != "" {
		if !IsValidProjectStatus(p.Status) {
			return ErrInvalidStatus(p.Status)
		}
	}

	if p.Progress != nil {
		if err := p.Progress.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the job input message
func (j *JobInputMessage) Validate() error {
	// platform is required - should be in meta.platform of payload items
	if j.Platform == "" {
		return ErrMissingRequiredField("platform")
	}

	if !IsValidPlatform(j.Platform) {
		return ErrInvalidPlatform(j.Platform)
	}

	// status is optional - Job status may not be at top level in message format
	// Message may only have success boolean, not Job.status
	// If status is provided, validate it; otherwise allow empty
	if j.Status != "" {
		if !IsValidJobStatus(j.Status) {
			return ErrInvalidStatus(j.Status)
		}
	}

	if j.Progress != nil {
		if err := j.Progress.Validate(); err != nil {
			return err
		}
	}

	if j.Batch != nil {
		if err := j.Batch.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the progress input
func (p *ProgressInput) Validate() error {
	if p.Current < 0 {
		return ErrInvalidValue("current", "must be non-negative")
	}

	if p.Total < 0 {
		return ErrInvalidValue("total", "must be non-negative")
	}

	if p.Current > p.Total {
		return ErrInvalidValue("current", "cannot exceed total")
	}

	if p.Percentage < 0 || p.Percentage > 100 {
		return ErrInvalidValue("percentage", "must be between 0 and 100")
	}

	if p.ETA < 0 {
		return ErrInvalidValue("eta", "must be non-negative")
	}

	return nil
}

// Validate validates the batch input
func (b *BatchInput) Validate() error {
	if b.Keyword == "" {
		return ErrMissingRequiredField("keyword")
	}

	if b.CrawledAt == "" {
		return ErrMissingRequiredField("crawled_at")
	}

	// Validate each content item
	for i, content := range b.ContentList {
		if err := content.Validate(); err != nil {
			return ErrInvalidArrayItem("content_list", i, err)
		}
	}

	return nil
}

// Validate validates the content input
func (c *ContentInput) Validate() error {
	if c.ID == "" {
		return ErrMissingRequiredField("id")
	}

	if c.Text == "" {
		return ErrMissingRequiredField("text")
	}

	if err := c.Author.Validate(); err != nil {
		return ErrInvalidField("author", err)
	}

	if err := c.Metrics.Validate(); err != nil {
		return ErrInvalidField("metrics", err)
	}

	if c.Media != nil {
		if err := c.Media.Validate(); err != nil {
			return ErrInvalidField("media", err)
		}
	}

	if c.PublishedAt == "" {
		return ErrMissingRequiredField("published_at")
	}

	if c.Permalink == "" {
		return ErrMissingRequiredField("permalink")
	}

	return nil
}

// Validate validates the author input
func (a *AuthorInput) Validate() error {
	if a.ID == "" {
		return ErrMissingRequiredField("id")
	}

	if a.Username == "" {
		return ErrMissingRequiredField("username")
	}

	if a.Name == "" {
		return ErrMissingRequiredField("name")
	}

	if a.Followers < 0 {
		return ErrInvalidValue("followers", "must be non-negative")
	}

	// avatar_url is optional - allow empty string
	// If missing, UI can use a default placeholder avatar

	return nil
}

// Validate validates the metrics input
func (m *MetricsInput) Validate() error {
	if m.Views < 0 {
		return ErrInvalidValue("views", "must be non-negative")
	}

	if m.Likes < 0 {
		return ErrInvalidValue("likes", "must be non-negative")
	}

	if m.Comments < 0 {
		return ErrInvalidValue("comments", "must be non-negative")
	}

	if m.Shares < 0 {
		return ErrInvalidValue("shares", "must be non-negative")
	}

	if m.Rate < 0 {
		return ErrInvalidValue("rate", "must be non-negative")
	}

	return nil
}

// Validate validates the media input
func (m *MediaInput) Validate() error {
	if m.Type == "" {
		return ErrMissingRequiredField("type")
	}

	// Validate media type
	validTypes := []string{"video", "image", "audio"}
	isValid := false
	for _, vt := range validTypes {
		if m.Type == vt {
			isValid = true
			break
		}
	}

	if !isValid {
		return ErrInvalidValue("type", "must be video, image, or audio")
	}

	if m.Duration < 0 {
		return ErrInvalidValue("duration", "must be non-negative")
	}

	// thumbnail is optional - crawler may not provide it (currently always null)
	// UI can use a default placeholder thumbnail if missing

	// url is optional - crawler may only provide video_path/audio_path (MinIO path)
	// not public URL. UI can construct URL from path if needed.

	return nil
}

// ToJSON converts the input message to JSON bytes
func (p *ProjectInputMessage) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// ToJSON converts the input message to JSON bytes
func (j *JobInputMessage) ToJSON() ([]byte, error) {
	return json.Marshal(j)
}

// ============================================================================
// Phase-Based Progress Types (New Format)
// ============================================================================

// PhaseProgressInput represents progress for a single phase (crawl/analyze)
type PhaseProgressInput struct {
	Total           int64   `json:"total"`            // Total items in this phase
	Done            int64   `json:"done"`             // Completed items in this phase
	Errors          int64   `json:"errors"`           // Failed items in this phase
	ProgressPercent float64 `json:"progress_percent"` // Phase progress (0.0-100.0)
}

// ProjectPhaseInputMessage represents the NEW input structure with phase-based progress
// Channel format: project:{project_id}:{user_id}
type ProjectPhaseInputMessage struct {
	Type    string                   `json:"type"`    // "project_progress" or "project_completed"
	Payload ProjectPhasePayloadInput `json:"payload"` // Message payload
}

// ProjectPhasePayloadInput represents the payload for phase-based project messages
type ProjectPhasePayloadInput struct {
	ProjectID              string              `json:"project_id"`               // Project identifier
	Status                 string              `json:"status"`                   // "INITIALIZING", "PROCESSING", "DONE", "FAILED"
	Crawl                  *PhaseProgressInput `json:"crawl,omitempty"`          // Progress for crawl phase
	Analyze                *PhaseProgressInput `json:"analyze,omitempty"`        // Progress for analyze phase
	OverallProgressPercent float64             `json:"overall_progress_percent"` // Combined progress (0.0-100.0)
}

// Validate validates the phase progress input
func (p *PhaseProgressInput) Validate() error {
	if p.Total < 0 {
		return ErrInvalidValue("total", "must be non-negative")
	}
	if p.Done < 0 {
		return ErrInvalidValue("done", "must be non-negative")
	}
	if p.Done > p.Total && p.Total > 0 {
		return ErrInvalidValue("done", "cannot exceed total")
	}
	if p.Errors < 0 {
		return ErrInvalidValue("errors", "must be non-negative")
	}
	if p.ProgressPercent < 0 || p.ProgressPercent > 100 {
		return ErrInvalidValue("progress_percent", "must be between 0 and 100")
	}
	return nil
}

// Validate validates the project phase input message
func (p *ProjectPhaseInputMessage) Validate() error {
	validTypes := []string{"project_progress", "project_completed"}
	isValidType := false
	for _, t := range validTypes {
		if p.Type == t {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return ErrInvalidValue("type", "must be project_progress or project_completed")
	}
	return p.Payload.Validate()
}

// Validate validates the project phase payload input
func (p *ProjectPhasePayloadInput) Validate() error {
	if p.ProjectID == "" {
		return ErrMissingRequiredField("project_id")
	}

	if !IsValidPhaseBasedProjectStatus(p.Status) {
		return ErrInvalidStatus(p.Status)
	}

	if p.Crawl != nil {
		if err := p.Crawl.Validate(); err != nil {
			return ErrInvalidField("crawl", err)
		}
	}

	if p.Analyze != nil {
		if err := p.Analyze.Validate(); err != nil {
			return ErrInvalidField("analyze", err)
		}
	}

	if p.OverallProgressPercent < 0 || p.OverallProgressPercent > 100 {
		return ErrInvalidValue("overall_progress_percent", "must be between 0 and 100")
	}

	return nil
}

// ToJSON converts the project phase input message to JSON bytes
func (p *ProjectPhaseInputMessage) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// IsPhaseBasedMessage checks if the payload is in phase-based format
func IsPhaseBasedMessage(payload []byte) bool {
	var msg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return false
	}
	return msg.Type == "project_progress" || msg.Type == "project_completed"
}

// ============================================================================
// Output Types (from internal/types/output.go)
// ============================================================================

// ProjectNotificationMessage represents the standardized output structure for project notifications
type ProjectNotificationMessage struct {
	Status   ProjectStatus `json:"status"`             // Current project status (enum)
	Progress *Progress     `json:"progress,omitempty"` // Overall progress (omit if empty)
}

// JobNotificationMessage represents the standardized output structure for job notifications
type JobNotificationMessage struct {
	Platform Platform   `json:"platform"`           // Social media platform enum
	Status   JobStatus  `json:"status"`             // Current job processing status
	Batch    *BatchData `json:"batch,omitempty"`    // Current batch crawl results (omit if empty)
	Progress *Progress  `json:"progress,omitempty"` // Overall job progress statistics (omit if empty)
}

// Progress represents progress information in standardized output format
type Progress struct {
	Current    int      `json:"current"`    // Current completed items
	Total      int      `json:"total"`      // Total items to process
	Percentage float64  `json:"percentage"` // Completion percentage (0-100)
	ETA        float64  `json:"eta"`        // Estimated time remaining in minutes
	Errors     []string `json:"errors"`     // Array of error messages encountered
}

// BatchData represents a batch of crawled content in standardized output format
type BatchData struct {
	Keyword     string        `json:"keyword"`      // Search keyword for this batch
	ContentList []ContentItem `json:"content_list"` // Crawled content items
	CrawledAt   string        `json:"crawled_at"`   // When this batch was processed (ISO timestamp)
}

// ContentItem represents a single social media content item in standardized output format
type ContentItem struct {
	ID          string            `json:"id"`              // Content unique ID
	Text        string            `json:"text"`            // Content text/caption
	Author      AuthorInfo        `json:"author"`          // Author information
	Metrics     EngagementMetrics `json:"metrics"`         // Engagement statistics
	Media       *MediaInfo        `json:"media,omitempty"` // Media information (if any)
	PublishedAt string            `json:"published_at"`    // When content was published (ISO timestamp)
	Permalink   string            `json:"permalink"`       // Direct link to content
}

// AuthorInfo represents content author details in standardized output format
type AuthorInfo struct {
	ID         string `json:"id"`          // Author unique ID
	Username   string `json:"username"`    // Author username/handle
	Name       string `json:"name"`        // Author display name
	Followers  int    `json:"followers"`   // Follower count
	IsVerified bool   `json:"is_verified"` // Verification status
	AvatarURL  string `json:"avatar_url"`  // Profile picture URL
}

// EngagementMetrics represents content engagement statistics in standardized output format
type EngagementMetrics struct {
	Views    int     `json:"views"`    // View count
	Likes    int     `json:"likes"`    // Like count
	Comments int     `json:"comments"` // Comment count
	Shares   int     `json:"shares"`   // Share count
	Rate     float64 `json:"rate"`     // Engagement rate percentage
}

// MediaInfo represents media content details in standardized output format
type MediaInfo struct {
	Type      string `json:"type"`               // "video", "image", "audio"
	Duration  int    `json:"duration,omitempty"` // Duration in seconds (for video/audio)
	Thumbnail string `json:"thumbnail"`          // Thumbnail/preview URL
	URL       string `json:"url"`                // Media file URL
}

// Validate validates the project notification message
func (p *ProjectNotificationMessage) Validate() error {
	// status is optional - Project status may not be in message format
	// Collector Service manages status in Redis, may not push to crawler
	// If status is provided, validate it; otherwise allow empty
	if p.Status != "" {
		if !IsValidProjectStatus(string(p.Status)) {
			return ErrInvalidStatus(string(p.Status))
		}
	}

	if p.Progress != nil {
		if err := p.Progress.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the job notification message
func (j *JobNotificationMessage) Validate() error {
	// platform is required - should be in meta.platform of payload items
	if j.Platform == "" {
		return ErrMissingRequiredField("platform")
	}

	if !IsValidPlatform(string(j.Platform)) {
		return ErrInvalidPlatform(string(j.Platform))
	}

	// status is optional - Job status may not be at top level in message format
	// Message may only have success boolean, not Job.status
	// If status is provided, validate it; otherwise allow empty
	if j.Status != "" {
		if !IsValidJobStatus(string(j.Status)) {
			return ErrInvalidStatus(string(j.Status))
		}
	}

	if j.Progress != nil {
		if err := j.Progress.Validate(); err != nil {
			return err
		}
	}

	if j.Batch != nil {
		if err := j.Batch.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the progress
func (p *Progress) Validate() error {
	if p.Current < 0 {
		return ErrInvalidValue("current", "must be non-negative")
	}

	if p.Total < 0 {
		return ErrInvalidValue("total", "must be non-negative")
	}

	if p.Current > p.Total {
		return ErrInvalidValue("current", "cannot exceed total")
	}

	if p.Percentage < 0 || p.Percentage > 100 {
		return ErrInvalidValue("percentage", "must be between 0 and 100")
	}

	if p.ETA < 0 {
		return ErrInvalidValue("eta", "must be non-negative")
	}

	return nil
}

// Validate validates the batch data
func (b *BatchData) Validate() error {
	if b.Keyword == "" {
		return ErrMissingRequiredField("keyword")
	}

	if b.CrawledAt == "" {
		return ErrMissingRequiredField("crawled_at")
	}

	// Validate each content item
	for i, content := range b.ContentList {
		if err := content.Validate(); err != nil {
			return ErrInvalidArrayItem("content_list", i, err)
		}
	}

	return nil
}

// Validate validates the content item
func (c *ContentItem) Validate() error {
	if c.ID == "" {
		return ErrMissingRequiredField("id")
	}

	if c.Text == "" {
		return ErrMissingRequiredField("text")
	}

	if err := c.Author.Validate(); err != nil {
		return ErrInvalidField("author", err)
	}

	if err := c.Metrics.Validate(); err != nil {
		return ErrInvalidField("metrics", err)
	}

	if c.Media != nil {
		if err := c.Media.Validate(); err != nil {
			return ErrInvalidField("media", err)
		}
	}

	if c.PublishedAt == "" {
		return ErrMissingRequiredField("published_at")
	}

	if c.Permalink == "" {
		return ErrMissingRequiredField("permalink")
	}

	return nil
}

// Validate validates the author info
func (a *AuthorInfo) Validate() error {
	if a.ID == "" {
		return ErrMissingRequiredField("id")
	}

	if a.Username == "" {
		return ErrMissingRequiredField("username")
	}

	if a.Name == "" {
		return ErrMissingRequiredField("name")
	}

	if a.Followers < 0 {
		return ErrInvalidValue("followers", "must be non-negative")
	}

	// avatar_url is optional - allow empty string
	// If missing, UI can use a default placeholder avatar

	return nil
}

// Validate validates the engagement metrics
func (e *EngagementMetrics) Validate() error {
	if e.Views < 0 {
		return ErrInvalidValue("views", "must be non-negative")
	}

	if e.Likes < 0 {
		return ErrInvalidValue("likes", "must be non-negative")
	}

	if e.Comments < 0 {
		return ErrInvalidValue("comments", "must be non-negative")
	}

	if e.Shares < 0 {
		return ErrInvalidValue("shares", "must be non-negative")
	}

	if e.Rate < 0 {
		return ErrInvalidValue("rate", "must be non-negative")
	}

	return nil
}

// Validate validates the media info
func (m *MediaInfo) Validate() error {
	if m.Type == "" {
		return ErrMissingRequiredField("type")
	}

	// Validate media type
	validTypes := []string{"video", "image", "audio"}
	isValid := false
	for _, vt := range validTypes {
		if m.Type == vt {
			isValid = true
			break
		}
	}

	if !isValid {
		return ErrInvalidValue("type", "must be video, image, or audio")
	}

	if m.Duration < 0 {
		return ErrInvalidValue("duration", "must be non-negative")
	}

	// thumbnail is optional - crawler may not provide it (currently always null)
	// UI can use a default placeholder thumbnail if missing

	// url is optional - crawler may only provide video_path/audio_path (MinIO path)
	// not public URL. UI can construct URL from path if needed.

	return nil
}

// ToJSON converts the project notification message to JSON bytes
func (p *ProjectNotificationMessage) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// ToJSON converts the job notification message to JSON bytes
func (j *JobNotificationMessage) ToJSON() ([]byte, error) {
	return json.Marshal(j)
}

// ============================================================================
// Phase-Based Progress Types (New Format)
// ============================================================================

// PhaseProgress represents progress for a single phase in output format
type PhaseProgress struct {
	Total           int64   `json:"total"`            // Total items in this phase
	Done            int64   `json:"done"`             // Completed items in this phase
	Errors          int64   `json:"errors"`           // Failed items in this phase
	ProgressPercent float64 `json:"progress_percent"` // Phase progress (0.0-100.0)
}

// ProjectPhaseNotificationMessage represents the NEW output structure with phase-based progress
// This is sent directly to WebSocket clients
type ProjectPhaseNotificationMessage struct {
	Type    string                    `json:"type"`    // "project_progress" or "project_completed"
	Payload ProjectPhasePayloadOutput `json:"payload"` // Message payload
}

// ProjectPhasePayloadOutput represents the payload for phase-based project messages
type ProjectPhasePayloadOutput struct {
	ProjectID              string         `json:"project_id"`               // Project identifier
	Status                 string         `json:"status"`                   // "INITIALIZING", "PROCESSING", "DONE", "FAILED"
	Crawl                  *PhaseProgress `json:"crawl,omitempty"`          // Progress for crawl phase
	Analyze                *PhaseProgress `json:"analyze,omitempty"`        // Progress for analyze phase
	OverallProgressPercent float64        `json:"overall_progress_percent"` // Combined progress (0.0-100.0)
}

// Validate validates the phase progress output
func (p *PhaseProgress) Validate() error {
	if p.Total < 0 {
		return ErrInvalidValue("total", "must be non-negative")
	}
	if p.Done < 0 {
		return ErrInvalidValue("done", "must be non-negative")
	}
	if p.Errors < 0 {
		return ErrInvalidValue("errors", "must be non-negative")
	}
	if p.ProgressPercent < 0 || p.ProgressPercent > 100 {
		return ErrInvalidValue("progress_percent", "must be between 0 and 100")
	}
	return nil
}

// Validate validates the project phase notification message
func (p *ProjectPhaseNotificationMessage) Validate() error {
	validTypes := []string{"project_progress", "project_completed"}
	isValidType := false
	for _, t := range validTypes {
		if p.Type == t {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return ErrInvalidValue("type", "must be project_progress or project_completed")
	}
	return p.Payload.Validate()
}

// Validate validates the project phase payload output
func (p *ProjectPhasePayloadOutput) Validate() error {
	if p.ProjectID == "" {
		return ErrMissingRequiredField("project_id")
	}

	if p.Crawl != nil {
		if err := p.Crawl.Validate(); err != nil {
			return ErrInvalidField("crawl", err)
		}
	}

	if p.Analyze != nil {
		if err := p.Analyze.Validate(); err != nil {
			return ErrInvalidField("analyze", err)
		}
	}

	if p.OverallProgressPercent < 0 || p.OverallProgressPercent > 100 {
		return ErrInvalidValue("overall_progress_percent", "must be between 0 and 100")
	}

	return nil
}

// ToJSON converts the project phase notification message to JSON bytes
func (p *ProjectPhaseNotificationMessage) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}
