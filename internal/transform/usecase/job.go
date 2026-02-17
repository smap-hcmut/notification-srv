package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"notification-srv/internal/transform"
)

// Transform transforms job input message to standardized output format
func (t *jobTransformer) Transform(ctx context.Context, payload string, jobID, userID string) (*transform.JobNotificationMessage, error) {
	startTime := time.Now()

	// Validate input first
	if err := t.validator.ValidateJobInput(payload); err != nil {
		t.metrics.IncrementTransformError("job", "validation")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Parse input message
	var inputMsg transform.JobInputMessage
	if err := json.Unmarshal([]byte(payload), &inputMsg); err != nil {
		t.metrics.IncrementTransformError("job", "json_parse")
		return nil, fmt.Errorf("failed to parse input message: %w", err)
	}

	// Transform to output format
	outputMsg, err := t.transformToOutput(&inputMsg)
	if err != nil {
		t.metrics.IncrementTransformError("job", "transform")
		return nil, fmt.Errorf("failed to transform message: %w", err)
	}

	// Validate output
	if err := outputMsg.Validate(); err != nil {
		t.metrics.IncrementTransformError("job", "output_validation")
		return nil, fmt.Errorf("output validation failed: %w", err)
	}

	// Record success metrics
	duration := time.Since(startTime)
	t.metrics.RecordTransformLatency("job", duration)
	t.metrics.IncrementTransformSuccess("job")

	t.logger.Debugf(ctx, "Successfully transformed job message for job %s, user %s in %v",
		jobID, userID, duration)

	return outputMsg, nil
}

// transformToOutput converts input message to output format
func (t *jobTransformer) transformToOutput(input *transform.JobInputMessage) (*transform.JobNotificationMessage, error) {
	// Create output message
	output := &transform.JobNotificationMessage{
		Platform: transform.Platform(input.Platform),
		Status:   transform.JobStatus(input.Status),
	}

	// Transform progress if present
	if input.Progress != nil {
		progress, err := t.transformProgress(input.Progress)
		if err != nil {
			return nil, fmt.Errorf("failed to transform progress: %w", err)
		}
		output.Progress = progress
	}

	// Transform batch if present
	if input.Batch != nil {
		batch, err := t.transformBatch(input.Batch)
		if err != nil {
			return nil, fmt.Errorf("failed to transform batch: %w", err)
		}
		output.Batch = batch
	}

	return output, nil
}

// transformProgress transforms progress input to output format
func (t *jobTransformer) transformProgress(input *transform.ProgressInput) (*transform.Progress, error) {
	// Clamp percentage to valid range
	percentage := input.Percentage
	if percentage < 0 {
		percentage = 0
		t.logger.Warnf(context.Background(), "Clamped negative percentage %f to 0", input.Percentage)
	} else if percentage > 100 {
		percentage = 100
		t.logger.Warnf(context.Background(), "Clamped excessive percentage %f to 100", input.Percentage)
	}

	// Ensure ETA is non-negative
	eta := input.ETA
	if eta < 0 {
		eta = 0
		t.logger.Warnf(context.Background(), "Clamped negative ETA %f to 0", input.ETA)
	}

	// Copy errors slice to prevent shared references
	var errors []string
	if input.Errors != nil {
		errors = make([]string, len(input.Errors))
		copy(errors, input.Errors)
	} else {
		errors = []string{}
	}

	// Create output progress
	return &transform.Progress{
		Current:    input.Current,
		Total:      input.Total,
		Percentage: percentage,
		ETA:        eta,
		Errors:     errors,
	}, nil
}

// transformBatch transforms batch input to output format
func (t *jobTransformer) transformBatch(input *transform.BatchInput) (*transform.BatchData, error) {
	// Transform content list
	contentList, err := t.transformContentList(input.ContentList)
	if err != nil {
		return nil, fmt.Errorf("failed to transform content list: %w", err)
	}

	return &transform.BatchData{
		Keyword:     input.Keyword,
		ContentList: contentList,
		CrawledAt:   input.CrawledAt,
	}, nil
}

// transformContentList transforms array of content input to output format
func (t *jobTransformer) transformContentList(inputList []transform.ContentInput) ([]transform.ContentItem, error) {
	outputList := make([]transform.ContentItem, 0, len(inputList))
	seenIDs := make(map[string]bool)

	for i, input := range inputList {
		// Check for duplicate IDs
		if seenIDs[input.ID] {
			t.logger.Warnf(context.Background(), "Duplicate content ID detected and skipped: %s", input.ID)
			continue
		}
		seenIDs[input.ID] = true

		// Transform individual content item
		contentItem, err := t.transformContentItem(&input)
		if err != nil {
			return nil, fmt.Errorf("failed to transform content item at index %d: %w", i, err)
		}

		outputList = append(outputList, *contentItem)
	}

	return outputList, nil
}

// transformContentItem transforms content input to output format
func (t *jobTransformer) transformContentItem(input *transform.ContentInput) (*transform.ContentItem, error) {
	// Transform author
	author := transform.AuthorInfo{
		ID:         input.Author.ID,
		Username:   input.Author.Username,
		Name:       input.Author.Name,
		Followers:  input.Author.Followers,
		IsVerified: input.Author.IsVerified,
		AvatarURL:  input.Author.AvatarURL,
	}

	// Transform metrics
	metrics := transform.EngagementMetrics{
		Views:    input.Metrics.Views,
		Likes:    input.Metrics.Likes,
		Comments: input.Metrics.Comments,
		Shares:   input.Metrics.Shares,
		Rate:     input.Metrics.Rate,
	}

	// Create output content item
	contentItem := &transform.ContentItem{
		ID:          input.ID,
		Text:        input.Text,
		Author:      author,
		Metrics:     metrics,
		PublishedAt: input.PublishedAt,
		Permalink:   input.Permalink,
	}

	// Transform media if present
	if input.Media != nil {
		media := &transform.MediaInfo{
			Type:      input.Media.Type,
			Duration:  input.Media.Duration,
			Thumbnail: input.Media.Thumbnail,
			URL:       input.Media.URL,
		}
		contentItem.Media = media
	}

	return contentItem, nil
}

// deduplicateContentByID removes duplicate content items based on ID, keeping first occurrence
func (t *jobTransformer) deduplicateContentByID(inputList []transform.ContentInput) []transform.ContentInput {
	seen := make(map[string]bool)
	result := make([]transform.ContentInput, 0, len(inputList))
	duplicateCount := 0

	for _, item := range inputList {
		if !seen[item.ID] {
			seen[item.ID] = true
			result = append(result, item)
		} else {
			duplicateCount++
			t.logger.Warnf(context.Background(), "Removing duplicate content item with ID: %s", item.ID)
		}
	}

	if duplicateCount > 0 {
		t.logger.Infof(context.Background(), "Removed %d duplicate content items from batch", duplicateCount)
	}

	return result
}
