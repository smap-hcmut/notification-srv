package types

import (
	"encoding/json"
	"testing"
)

func TestProjectInputMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		msg     ProjectInputMessage
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message with progress",
			msg: ProjectInputMessage{
				Status: "PROCESSING",
				Progress: &ProgressInput{
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
			msg: ProjectInputMessage{
				Status: "COMPLETED",
			},
			wantErr: false,
		},
		{
			name: "missing status",
			msg: ProjectInputMessage{
				Progress: &ProgressInput{},
			},
			wantErr: true,
			errMsg:  "missing required field: status",
		},
		{
			name: "invalid status",
			msg: ProjectInputMessage{
				Status: "UNKNOWN",
			},
			wantErr: true,
			errMsg:  "invalid status: UNKNOWN",
		},
		{
			name: "invalid progress",
			msg: ProjectInputMessage{
				Status: "PROCESSING",
				Progress: &ProgressInput{
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
				t.Errorf("ProjectInputMessage.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("ProjectInputMessage.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestJobInputMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		msg     JobInputMessage
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid job message",
			msg: JobInputMessage{
				Platform: "TIKTOK",
				Status:   "PROCESSING",
				Progress: &ProgressInput{
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
			name: "missing platform",
			msg: JobInputMessage{
				Status: "PROCESSING",
			},
			wantErr: true,
			errMsg:  "missing required field: platform",
		},
		{
			name: "invalid platform",
			msg: JobInputMessage{
				Platform: "FACEBOOK",
				Status:   "PROCESSING",
			},
			wantErr: true,
			errMsg:  "invalid platform: FACEBOOK",
		},
		{
			name: "missing status",
			msg: JobInputMessage{
				Platform: "TIKTOK",
			},
			wantErr: true,
			errMsg:  "missing required field: status",
		},
		{
			name: "invalid status",
			msg: JobInputMessage{
				Platform: "TIKTOK",
				Status:   "UNKNOWN",
			},
			wantErr: true,
			errMsg:  "invalid status: UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("JobInputMessage.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("JobInputMessage.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestProgressInput_Validate(t *testing.T) {
	tests := []struct {
		name     string
		progress ProgressInput
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid progress",
			progress: ProgressInput{
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
			progress: ProgressInput{
				Current: -1,
				Total:   100,
			},
			wantErr: true,
			errMsg:  "invalid value for current: must be non-negative",
		},
		{
			name: "negative total",
			progress: ProgressInput{
				Current: 50,
				Total:   -1,
			},
			wantErr: true,
			errMsg:  "invalid value for total: must be non-negative",
		},
		{
			name: "current exceeds total",
			progress: ProgressInput{
				Current: 150,
				Total:   100,
			},
			wantErr: true,
			errMsg:  "invalid value for current: cannot exceed total",
		},
		{
			name: "percentage below 0",
			progress: ProgressInput{
				Current:    50,
				Total:      100,
				Percentage: -1.0,
			},
			wantErr: true,
			errMsg:  "invalid value for percentage: must be between 0 and 100",
		},
		{
			name: "percentage above 100",
			progress: ProgressInput{
				Current:    50,
				Total:      100,
				Percentage: 101.0,
			},
			wantErr: true,
			errMsg:  "invalid value for percentage: must be between 0 and 100",
		},
		{
			name: "negative ETA",
			progress: ProgressInput{
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
				t.Errorf("ProgressInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("ProgressInput.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestBatchInput_Validate(t *testing.T) {
	validContent := ContentInput{
		ID:          "123",
		Text:        "Test content",
		Author:      AuthorInput{ID: "author1", Username: "user1", Name: "User", Followers: 100, AvatarURL: "http://example.com"},
		Metrics:     MetricsInput{Views: 1000, Likes: 50, Comments: 5, Shares: 2, Rate: 5.0},
		PublishedAt: "2024-01-01T00:00:00Z",
		Permalink:   "http://example.com/content/123",
	}

	tests := []struct {
		name    string
		batch   BatchInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid batch",
			batch: BatchInput{
				Keyword:     "test keyword",
				ContentList: []ContentInput{validContent},
				CrawledAt:   "2024-01-01T00:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "missing keyword",
			batch: BatchInput{
				ContentList: []ContentInput{validContent},
				CrawledAt:   "2024-01-01T00:00:00Z",
			},
			wantErr: true,
			errMsg:  "missing required field: keyword",
		},
		{
			name: "missing crawled_at",
			batch: BatchInput{
				Keyword:     "test keyword",
				ContentList: []ContentInput{validContent},
			},
			wantErr: true,
			errMsg:  "missing required field: crawled_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.batch.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BatchInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("BatchInput.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestContentInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		content ContentInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid content",
			content: ContentInput{
				ID:          "123",
				Text:        "Test content",
				Author:      AuthorInput{ID: "author1", Username: "user1", Name: "User", Followers: 100, AvatarURL: "http://example.com"},
				Metrics:     MetricsInput{Views: 1000, Likes: 50, Comments: 5, Shares: 2, Rate: 5.0},
				PublishedAt: "2024-01-01T00:00:00Z",
				Permalink:   "http://example.com/content/123",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			content: ContentInput{
				Text:        "Test content",
				Author:      AuthorInput{ID: "author1", Username: "user1", Name: "User", Followers: 100, AvatarURL: "http://example.com"},
				Metrics:     MetricsInput{Views: 1000, Likes: 50, Comments: 5, Shares: 2, Rate: 5.0},
				PublishedAt: "2024-01-01T00:00:00Z",
				Permalink:   "http://example.com/content/123",
			},
			wantErr: true,
			errMsg:  "missing required field: id",
		},
		{
			name: "missing text",
			content: ContentInput{
				ID:          "123",
				Author:      AuthorInput{ID: "author1", Username: "user1", Name: "User", Followers: 100, AvatarURL: "http://example.com"},
				Metrics:     MetricsInput{Views: 1000, Likes: 50, Comments: 5, Shares: 2, Rate: 5.0},
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
				t.Errorf("ContentInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("ContentInput.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestMediaInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		media   MediaInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid video media",
			media: MediaInput{
				Type:      "video",
				Duration:  30,
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/video.mp4",
			},
			wantErr: false,
		},
		{
			name: "valid image media",
			media: MediaInput{
				Type:      "image",
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/image.jpg",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			media: MediaInput{
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/video.mp4",
			},
			wantErr: true,
			errMsg:  "missing required field: type",
		},
		{
			name: "invalid type",
			media: MediaInput{
				Type:      "document",
				Thumbnail: "http://example.com/thumb.jpg",
				URL:       "http://example.com/doc.pdf",
			},
			wantErr: true,
			errMsg:  "invalid value for type: must be video, image, or audio",
		},
		{
			name: "negative duration",
			media: MediaInput{
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
				t.Errorf("MediaInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("MediaInput.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestProjectInputMessage_ToJSON(t *testing.T) {
	msg := ProjectInputMessage{
		Status: "PROCESSING",
		Progress: &ProgressInput{
			Current:    50,
			Total:      100,
			Percentage: 50.0,
			ETA:        10.5,
			Errors:     []string{"error1", "error2"},
		},
	}

	data, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("ProjectInputMessage.ToJSON() error = %v", err)
	}

	// Verify JSON is valid
	var parsed ProjectInputMessage
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

func TestJobInputMessage_ToJSON(t *testing.T) {
	msg := JobInputMessage{
		Platform: "TIKTOK",
		Status:   "PROCESSING",
		Progress: &ProgressInput{
			Current:    10,
			Total:      50,
			Percentage: 20.0,
			ETA:        25.5,
			Errors:     []string{},
		},
	}

	data, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("JobInputMessage.ToJSON() error = %v", err)
	}

	// Verify JSON is valid
	var parsed JobInputMessage
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