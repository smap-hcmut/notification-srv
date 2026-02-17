package usecase

import (
	"context"
	"fmt"
	"time"

	"notification-srv/internal/alert"
	"notification-srv/pkg/discord"
)

func (uc *implUseCase) DispatchCampaignEvent(ctx context.Context, input alert.CampaignEventInput) error {
	fields := []discord.EmbedField{
		buildField("Event Type", input.EventType, true),
		buildField("Campaign", input.CampaignName, true),
		buildField("User", input.User, true),
	}

	if input.ResourceName != "" {
		val := input.ResourceName
		if input.ResourceURL != "" {
			val = fmt.Sprintf("[%s](%s)", input.ResourceName, input.ResourceURL)
		}
		fields = append(fields, buildField("Resource", val, false))
	}

	if input.Message != "" {
		fields = append(fields, buildField("Message", input.Message, false))
	}

	opts := discord.MessageOptions{
		Type:        discord.MessageTypeInfo,
		Title:       fmt.Sprintf("Campaign Event: %s", input.CampaignName),
		Description: fmt.Sprintf("Activity detected in campaign **%s** (%s).", input.CampaignName, input.CampaignID),
		Fields:      fields,
		Timestamp:   time.Now(),
		Footer: &discord.EmbedFooter{
			Text: "Notification Service • Campaign Manager",
		},
	}

	return uc.discord.SendEmbed(ctx, opts)
}
