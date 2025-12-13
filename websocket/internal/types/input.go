package types

import "encoding/json"

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
	Current    int      `json:"current"`     // Current completed items
	Total      int      `json:"total"`       // Total items to process
	Percentage float64  `json:"percentage"`  // Completion percentage (0-100)
	ETA        float64  `json:"eta"`         // Estimated time remaining in minutes
	Errors     []string `json:"errors"`      // Array of error messages
}

// BatchInput represents a batch of crawled content from publishers
type BatchInput struct {
	Keyword     string         `json:"keyword"`      // Search keyword
	ContentList []ContentInput `json:"content_list"` // Crawled content items
	CrawledAt   string         `json:"crawled_at"`   // ISO timestamp
}

// ContentInput represents a single social media content item from publishers
type ContentInput struct {
	ID          string       `json:"id"`           // Content unique ID
	Text        string       `json:"text"`         // Content text/caption
	Author      AuthorInput  `json:"author"`       // Author information
	Metrics     MetricsInput `json:"metrics"`      // Engagement statistics
	Media       *MediaInput  `json:"media,omitempty"`       // Media information
	PublishedAt string       `json:"published_at"` // ISO timestamp
	Permalink   string       `json:"permalink"`    // Direct link to content
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
	Type      string `json:"type"`                 // "video", "image", "audio"
	Duration  int    `json:"duration,omitempty"`   // Duration in seconds
	Thumbnail string `json:"thumbnail"`            // Thumbnail/preview URL
	URL       string `json:"url"`                  // Media file URL
}

// Validate validates the project input message
func (p *ProjectInputMessage) Validate() error {
	if p.Status == "" {
		return ErrMissingRequiredField("status")
	}
	
	if !IsValidProjectStatus(p.Status) {
		return ErrInvalidStatus(p.Status)
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
	if j.Platform == "" {
		return ErrMissingRequiredField("platform")
	}
	
	if !IsValidPlatform(j.Platform) {
		return ErrInvalidPlatform(j.Platform)
	}
	
	if j.Status == "" {
		return ErrMissingRequiredField("status")
	}
	
	if !IsValidJobStatus(j.Status) {
		return ErrInvalidStatus(j.Status)
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
	
	if a.AvatarURL == "" {
		return ErrMissingRequiredField("avatar_url")
	}
	
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
	
	if m.Thumbnail == "" {
		return ErrMissingRequiredField("thumbnail")
	}
	
	if m.URL == "" {
		return ErrMissingRequiredField("url")
	}
	
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