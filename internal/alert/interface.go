package alert

import "context"

// UseCase defines the logic for dispatching alerts to external channels.
type UseCase interface {
	// DispatchCrisisAlert sends a high-priority alert to Discord.
	DispatchCrisisAlert(ctx context.Context, input CrisisAlertInput) error

	// DispatchDataOnboarding sends a status report for data ingestion.
	DispatchDataOnboarding(ctx context.Context, input DataOnboardingInput) error

	// DispatchCampaignEvent sends updates about campaign lifecycle events.
	DispatchCampaignEvent(ctx context.Context, input CampaignEventInput) error
}
