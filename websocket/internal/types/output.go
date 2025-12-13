package types

import "encoding/json"

// ProjectNotificationMessage represents the standardized output structure for project notifications
type ProjectNotificationMessage struct {
	Status   ProjectStatus `json:"status"`             // Current project status (enum)
	Progress *Progress     `json:"progress,omitempty"` // Overall progress (omit if empty)
}

// JobNotificationMessage represents the standardized output structure for job notifications
type JobNotificationMessage struct {
	Platform Platform    `json:"platform"`           // Social media platform enum
	Status   JobStatus   `json:"status"`             // Current job processing status
	Batch    *BatchData  `json:"batch,omitempty"`    // Current batch crawl results (omit if empty)
	Progress *Progress   `json:"progress,omitempty"` // Overall job progress statistics (omit if empty)
}

// Progress represents progress information in standardized output format
type Progress struct {
	Current    int      `json:"current"`     // Current completed items
	Total      int      `json:"total"`       // Total items to process
	Percentage float64  `json:"percentage"`  // Completion percentage (0-100)
	ETA        float64  `json:"eta"`         // Estimated time remaining in minutes
	Errors     []string `json:"errors"`      // Array of error messages encountered
}

// BatchData represents a batch of crawled content in standardized output format
type BatchData struct {
	Keyword     string        `json:"keyword"`      // Search keyword for this batch
	ContentList []ContentItem `json:"content_list"` // Crawled content items
	CrawledAt   string        `json:"crawled_at"`   // When this batch was processed (ISO timestamp)
}

// ContentItem represents a single social media content item in standardized output format
type ContentItem struct {
	ID          string            `json:"id"`           // Content unique ID
	Text        string            `json:"text"`         // Content text/caption
	Author      AuthorInfo        `json:"author"`       // Author information
	Metrics     EngagementMetrics `json:"metrics"`      // Engagement statistics
	Media       *MediaInfo        `json:"media,omitempty"`       // Media information (if any)
	PublishedAt string            `json:"published_at"` // When content was published (ISO timestamp)
	Permalink   string            `json:"permalink"`    // Direct link to content
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
	Type      string `json:"type"`                 // "video", "image", "audio"
	Duration  int    `json:"duration,omitempty"`   // Duration in seconds (for video/audio)
	Thumbnail string `json:"thumbnail"`            // Thumbnail/preview URL
	URL       string `json:"url"`                  // Media file URL
}

// Validate validates the project notification message
func (p *ProjectNotificationMessage) Validate() error {
	if p.Status == "" {
		return ErrMissingRequiredField("status")
	}
	
	if !IsValidProjectStatus(string(p.Status)) {
		return ErrInvalidStatus(string(p.Status))
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
	if j.Platform == "" {
		return ErrMissingRequiredField("platform")
	}
	
	if !IsValidPlatform(string(j.Platform)) {
		return ErrInvalidPlatform(string(j.Platform))
	}
	
	if j.Status == "" {
		return ErrMissingRequiredField("status")
	}
	
	if !IsValidJobStatus(string(j.Status)) {
		return ErrInvalidStatus(string(j.Status))
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
	
	if a.AvatarURL == "" {
		return ErrMissingRequiredField("avatar_url")
	}
	
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
	
	if m.Thumbnail == "" {
		return ErrMissingRequiredField("thumbnail")
	}
	
	if m.URL == "" {
		return ErrMissingRequiredField("url")
	}
	
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