package types

import (
	"encoding/json"
	"testing"
)

func TestProjectNotificationMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		msg     ProjectNotificationMessage
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message with progress",
			msg: ProjectNotificationMessage{
				Status: ProjectStatusProcessing,
				Progress: &Progress{
					Current:    50,
					Total:      100,
					Percentage: 50.0,
					ETA:        10.5,
					Errors:     []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "valid message without progress",
			msg: ProjectNotificationMessage{
				Status: ProjectStatusCompleted,
			},
			wantErr: false,
		},
		{
			name: "empty status",
			msg: ProjectNotificationMessage{
				Progress: &Progress{},
			},
			wantErr: true,
			errMsg:  "missing required field: status",
		},
		{
			name: "invalid progress",
			msg: ProjectNotificationMessage{
				Status: ProjectStatusProcessing,
				Progress: &Progress{
					Current:    -1,
					Total:      100,
					Percentage: 50.0,
				},
			},
			wantErr: true,
			errMsg:  "invalid value for current: must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProjectNotificationMessage.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("ProjectNotificationMessage.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestJobNotificationMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		msg     JobNotificationMessage
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid job message",
			msg: JobNotificationMessage{
				Platform: PlatformTikTok,
				Status:   JobStatusProcessing,
				Progress: &Progress{
					Current:    10,
					Total:      50,
					Percentage: 20.0,
					ETA:        30.0,
					Errors:     []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "empty platform",
			msg: JobNotificationMessage{
				Status: JobStatusProcessing,
			},
			wantErr: true,
			errMsg:  "missing required field: platform",
		},
		{
			name: "empty status",
			msg: JobNotificationMessage{
				Platform: PlatformTikTok,
			},
			wantErr: true,
			errMsg:  "missing required field: status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("JobNotificationMessage.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("JobNotificationMessage.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestProgress_Validate(t *testing.T) {
	tests := []struct {
		name     string
		progress Progress
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid progress",
			progress: Progress{
				Current:    75,
				Total:      100,
				Percentage: 75.0,
				ETA:        5.25,
				Errors:     []string{},
			},
			wantErr: false,
		},
		{
			name: "negative current",
			progress: Progress{
				Current: -1,
				Total:   100,
			},
			wantErr: true,
			errMsg:  "invalid value for current: must be non-negative",
		},
		{
			name: "negative total",
			progress: Progress{
				Current: 50,
				Total:   -1,
			},
			wantErr: true,
			errMsg:  "invalid value for total: must be non-negative",
		},
		{
			name: "current exceeds total",
			progress: Progress{
				Current: 150,
				Total:   100,
			},
			wantErr: true,
			errMsg:  "invalid value for current: cannot exceed total",
		},
		{
			name: "percentage below 0",
			progress: Progress{
				Current:    50,
				Total:      100,
				Percentage: -1.0,
			},
			wantErr: true,
			errMsg:  "invalid value for percentage: must be between 0 and 100",
		},
		{
			name: "percentage above 100",
			progress: Progress{
				Current:    50,
				Total:      100,
				Percentage: 101.0,
			},
			wantErr: true,
			errMsg:  "invalid value for percentage: must be between 0 and 100",
		},
		{
			name: "negative ETA",
			progress: Progress{
				Current:    50,
				Total:      100,
				Percentage: 50.0,
				ETA:        -1.0,
			},
			wantErr: true,
			errMsg:  "invalid value for eta: must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.progress.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Progress.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("Progress.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestBatchData_Validate(t *testing.T) {
	validContent := ContentItem{
		ID:          "123",
		Text:        "Test content",
		Author:      AuthorInfo{ID: "author1", Username: "user1", Name: "User", Followers: 100, AvatarURL: "http://example.com"},
		Metrics:     EngagementMetrics{Views: 1000, Likes: 50, Comments: 5, Shares: 2, Rate: 5.0},
		PublishedAt: "2024-01-01T00:00:00Z",
		Permalink:   "http://example.com/content/123",
	}

	tests := []struct {
		name    string
		batch   BatchData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid batch",
			batch: BatchData{
				Keyword:     "test keyword",
				ContentList: []ContentItem{validContent},
				CrawledAt:   "2024-01-01T00:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "missing keyword",
			batch: BatchData{
				ContentList: []ContentItem{validContent},
				CrawledAt:   "2024-01-01T00:00:00Z",
			},
			wantErr: true,
			errMsg:  "missing required field: keyword",
		},
		{
			name: "missing crawled_at",
			batch: BatchData{
				Keyword:     "test keyword",
				ContentList: []ContentItem{validContent},
			},
			wantErr: true,
			errMsg:  "missing required field: crawled_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.batch.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BatchData.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("BatchData.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestContentItem_Validate(t *testing.T) {
	tests := []struct {
		name    string
		content ContentItem
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid content",
			content: ContentItem{
				ID:          "123",
				Text:        "Test content",
				Author:      AuthorInfo{ID: "author1", Username: "user1", Name: "User", Followers: 100, AvatarURL: "http://example.com"},
				Metrics:     EngagementMetrics{Views: 1000, Likes: 50, Comments: 5, Shares: 2, Rate: 5.0},
				PublishedAt: "2024-01-01T00:00:00Z",
				Permalink:   "http://example.com/content/123",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			content: ContentItem{
				Text:        "Test content",
				Author:      AuthorInfo{ID: "author1", Username: "user1", Name: "User", Followers: 100, AvatarURL: "http://example.com"},
				Metrics:     EngagementMetrics{Views: 1000, Likes: 50, Comments: 5, Shares: 2, Rate: 5.0},
				PublishedAt: "2024-01-01T00:00:00Z",
				Permalink:   "http://example.com/content/123",
			},
			wantErr: true,
			errMsg:  "missing required field: id",
		},
		{
			name: "missing text",
			content: ContentItem{
				ID:          "123",
				Author:      AuthorInfo{ID: "author1", Username: "user1", Name: "User", Followers: 100, AvatarURL: "http://example.com"},
				Metrics:     EngagementMetrics{Views: 1000, Likes: 50, Comments: 5, Shares: 2, Rate: 5.0},
				PublishedAt: "2024-01-01T00:00:00Z",
				Permalink:   "http://example.com/content/123",
			},
			wantErr: true,
			errMsg:  "missing required field: text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.content.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ContentItem.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("ContentItem.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestMediaInfo_Validate(t *testing.T) {
	tests := []struct {
		name    string
		media   MediaInfo
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid video media",
			media: MediaInfo{
				Type:      "video",
				Duration:  30,
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/video.mp4",
			},
			wantErr: false,
		},
		{
			name: "valid image media",
			media: MediaInfo{
				Type:      "image",
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/image.jpg",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			media: MediaInfo{
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/video.mp4",
			},
			wantErr: true,
			errMsg:  "missing required field: type",
		},
		{
			name: "invalid type",
			media: MediaInfo{
				Type:      "document",
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/doc.pdf",
			},
			wantErr: true,
			errMsg:  "invalid value for type: must be video, image, or audio",
		},
		{
			name: "negative duration",
			media: MediaInfo{
				Type:      "video",
				Duration:  -1,
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/video.mp4",
			},
			wantErr: true,
			errMsg:  "invalid value for duration: must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.media.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MediaInfo.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("MediaInfo.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestProjectNotificationMessage_ToJSON(t *testing.T) {
	msg := ProjectNotificationMessage{
		Status: ProjectStatusProcessing,
		Progress: &Progress{
			Current:    50,
			Total:      100,
			Percentage: 50.0,
			ETA:        10.5,
			Errors:     []string{"error1", "error2"},
		},
	}

	data, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ProjectNotificationMessage.ToJSON() error = %v", err)
	}

	// Verify JSON is valid
	var parsed ProjectNotificationMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify content
	if parsed.Status != msg.Status {
		t.Errorf("Status = %v, want %v", parsed.Status, msg.Status)
	}
	if parsed.Progress.Current != msg.Progress.Current {
		t.Errorf("Progress.Current = %v, want %v", parsed.Progress.Current, msg.Progress.Current)
	}
}

func TestJobNotificationMessage_ToJSON(t *testing.T) {
	msg := JobNotificationMessage{
		Platform: PlatformTikTok,
		Status:   JobStatusProcessing,
		Progress: &Progress{
			Current:    10,
			Total:      50,
			Percentage: 20.0,
			ETA:        25.5,
			Errors:     []string{},
		},
	}

	data, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("JobNotificationMessage.ToJSON() error = %v", err)
	}

	// Verify JSON is valid
	var parsed JobNotificationMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify content
	if parsed.Platform != msg.Platform {
		t.Errorf("Platform = %v, want %v", parsed.Platform, msg.Platform)
	}
	if parsed.Status != msg.Status {
		t.Errorf("Status = %v, want %v", parsed.Status, msg.Status)
	}
}

func TestJSONOmitEmpty(t *testing.T) {
	// Test that omitempty works correctly
	msg := ProjectNotificationMessage{
		Status: ProjectStatusCompleted,
		// Progress is nil, should be omitted
	}

	data, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Check that progress field is not present in JSON
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := jsonMap["progress"]; exists {
		t.Error("progress field should be omitted when nil")
	}

	if jsonMap["status"] != string(ProjectStatusCompleted) {
		t.Errorf("status = %v, want %v", jsonMap["status"], string(ProjectStatusCompleted))
	}
}