package alert

import "context"

// UseCase defines the alert dispatching interface.
type UseCase interface {
	DispatchCrisisAlert(ctx context.Context, input CrisisAlertInput) error
	DispatchDataOnboarding(ctx context.Context, input DataOnboardingInput) error
	DispatchCampaignEvent(ctx context.Context, input CampaignEventInput) error
}

// Stub types for now
type CrisisAlertInput struct{}
type DataOnboardingInput struct{}
type CampaignEventInput struct{}
