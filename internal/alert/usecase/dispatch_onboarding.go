package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"notification-srv/internal/alert"
	"notification-srv/pkg/discord"
)

func (uc *implUseCase) DispatchDataOnboarding(ctx context.Context, input alert.DataOnboardingInput) error {
	// Only notify for final states or significant errors
	status := strings.ToLower(input.Status)
	if status != "completed" && status != "failed" {
		return nil
	}

	fields := []discord.EmbedField{
		buildField("Source", fmt.Sprintf("%s (%s)", input.SourceName, input.SourceType), true),
		buildField("Records Processed", fmt.Sprintf("%d", input.RecordCount), true),
		buildField("Errors", fmt.Sprintf("%d", input.ErrorCount), true),
		buildField("Duration", input.Duration.String(), true),
	}

	if input.Message != "" {
		fields = append(fields, buildField("Details", input.Message, false))
	}

	title := fmt.Sprintf("Data Onboarding: %s", strings.Title(status))
	desc := fmt.Sprintf("Data ingestion for **%s** has finished.", input.ProjectID)

	msgType := discord.MessageTypeSuccess
	if status == "failed" {
		msgType = discord.MessageTypeError
		title = fmt.Sprintf("Data Onboarding FAILED: %s", input.SourceName)
	}

	opts := discord.MessageOptions{
		Type:        msgType,
		Title:       title,
		Description: desc,
		Fields:      fields,
		Timestamp:   time.Now(),
		Footer: &discord.EmbedFooter{
			Text: "Notification Service • Data Pipeline",
		},
	}

	return uc.discord.SendEmbed(ctx, opts)
}
