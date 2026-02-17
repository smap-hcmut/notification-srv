package usecase

import (
	"context"

	"notification-srv/internal/alert"
	"notification-srv/pkg/discord"
	"notification-srv/pkg/log"
)

type implUseCase struct {
	logger  log.Logger
	discord discord.IDiscord
}

func New(logger log.Logger, discord discord.IDiscord) alert.UseCase {
	return &implUseCase{
		logger:  logger,
		discord: discord,
	}
}

func (uc *implUseCase) DispatchCrisisAlert(ctx context.Context, input alert.CrisisAlertInput) error {
	// Stub implementation
	uc.logger.Infof(ctx, "DispatchCrisisAlert called")
	return nil
}

func (uc *implUseCase) DispatchDataOnboarding(ctx context.Context, input alert.DataOnboardingInput) error {
	return nil
}

func (uc *implUseCase) DispatchCampaignEvent(ctx context.Context, input alert.CampaignEventInput) error {
	return nil
}
