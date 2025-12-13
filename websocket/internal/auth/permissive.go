package auth

import (
	"context"

	"smap-websocket/pkg/log"
)

// PermissiveAuthorizer always allows access (for backward compatibility)
// This is used when authorization is disabled or during gradual rollout
type PermissiveAuthorizer struct {
	logger log.Logger
}

// NewPermissiveAuthorizer creates a new PermissiveAuthorizer
func NewPermissiveAuthorizer(logger log.Logger) *PermissiveAuthorizer {
	return &PermissiveAuthorizer{
		logger: logger,
	}
}

// CanAccessProject always returns true (permissive mode)
func (pa *PermissiveAuthorizer) CanAccessProject(ctx context.Context, userID, projectID string) (bool, error) {
	pa.logger.Debugf(ctx, "Permissive authorization: allowing user %s access to project %s", userID, projectID)
	return true, nil
}

// CanAccessJob always returns true (permissive mode)
func (pa *PermissiveAuthorizer) CanAccessJob(ctx context.Context, userID, jobID string) (bool, error) {
	pa.logger.Debugf(ctx, "Permissive authorization: allowing user %s access to job %s", userID, jobID)
	return true, nil
}
