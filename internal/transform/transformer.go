package transform

import (
	"context"
	"fmt"

	"smap-websocket/internal/types"
)

// MessageTransformerImpl implements the MessageTransformer interface
type MessageTransformerImpl struct {
	projectTransformer *ProjectTransformer
	jobTransformer     *JobTransformer
	validator          MessageValidator
	errorHandler       ErrorHandler
	logger             Logger
}

// NewMessageTransformer creates a new message transformer
func NewMessageTransformer(
	validator MessageValidator,
	metrics MetricsCollector,
	errorHandler ErrorHandler,
	logger Logger,
) *MessageTransformerImpl {
	return &MessageTransformerImpl{
		projectTransformer: NewProjectTransformer(validator, metrics, logger),
		jobTransformer:     NewJobTransformer(validator, metrics, logger),
		validator:          validator,
		errorHandler:       errorHandler,
		logger:             logger,
	}
}

// TransformProjectMessage transforms a project input message to output format
func (t *MessageTransformerImpl) TransformProjectMessage(ctx context.Context, payload string, projectID, userID string) (*types.ProjectNotificationMessage, error) {
	channel := fmt.Sprintf("project:%s:%s", projectID, userID)
	
	result, err := t.projectTransformer.Transform(ctx, payload, projectID, userID)
	if err != nil {
		t.errorHandler.HandleTransformError(ctx, "project", channel, err, payload)
		return nil, fmt.Errorf("project message transform failed: %w", err)
	}
	
	return result, nil
}

// TransformJobMessage transforms a job input message to output format
func (t *MessageTransformerImpl) TransformJobMessage(ctx context.Context, payload string, jobID, userID string) (*types.JobNotificationMessage, error) {
	channel := fmt.Sprintf("job:%s:%s", jobID, userID)
	
	result, err := t.jobTransformer.Transform(ctx, payload, jobID, userID)
	if err != nil {
		t.errorHandler.HandleTransformError(ctx, "job", channel, err, payload)
		return nil, fmt.Errorf("job message transform failed: %w", err)
	}
	
	return result, nil
}

// TransformMessage transforms any message based on topic type
func (t *MessageTransformerImpl) TransformMessage(ctx context.Context, channel string, payload string) (interface{}, error) {
	// Validate and parse topic format
	topicType, id, userID, err := ValidateTopicFormat(channel)
	if err != nil {
		t.errorHandler.HandleValidationError(ctx, "unknown", channel, err, payload)
		return nil, fmt.Errorf("invalid topic format: %w", err)
	}
	
	// Transform based on topic type
	switch topicType {
	case "project":
		return t.TransformProjectMessage(ctx, payload, id, userID)
	case "job":
		return t.TransformJobMessage(ctx, payload, id, userID)
	default:
		err := fmt.Errorf("unsupported topic type: %s", topicType)
		t.errorHandler.HandleValidationError(ctx, topicType, channel, err, payload)
		return nil, err
	}
}