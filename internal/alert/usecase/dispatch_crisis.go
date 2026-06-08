package usecase

import (
	"context"
	"fmt"
	"notification-srv/internal/alert"
	"strings"
	"time"

	"github.com/smap-hcmut/shared-libs/go/discord"
)

func (uc *implUseCase) DispatchCrisisAlert(ctx context.Context, input alert.CrisisAlertInput) error {
	fields := []discord.EmbedField{
		buildField("Severity", strings.ToUpper(input.Severity), true),
		buildField("Alert Type", strings.ToTitle(input.AlertType), true),
		buildField("Metric", input.Metric, true),
		buildField("Value vs Threshold", fmt.Sprintf("**%s** / %s", formatFloat(input.CurrentValue), formatFloat(input.Threshold)), true),
		buildField("Time Window", input.TimeWindow, true),
		buildField("Action Required", input.ActionRequired, false),
	}

	if len(input.AffectedAspects) > 0 {
		fields = append(fields, buildField("Affected Aspects", strings.Join(input.AffectedAspects, ", "), false))
	}

	// Prefer SampleReferences (have URL + excerpt) over the legacy
	// SampleMentions text-only list. Each reference renders as a markdown
	// link Discord makes clickable, so the on-call reviewer can jump
	// straight to the original post instead of grepping by snippet.
	if len(input.SampleReferences) > 0 {
		count := 3
		if len(input.SampleReferences) < 3 {
			count = len(input.SampleReferences)
		}
		lines := make([]string, 0, count)
		for _, ref := range input.SampleReferences[:count] {
			excerpt := strings.TrimSpace(ref.ContentExcerpt)
			if excerpt == "" {
				excerpt = "(no excerpt)"
			}
			if url := strings.TrimSpace(ref.URL); url != "" {
				lines = append(lines, fmt.Sprintf("> [%s](%s)", excerpt, url))
			} else {
				lines = append(lines, fmt.Sprintf("> %s", excerpt))
			}
		}
		fields = append(fields, buildField("Sample Mentions", strings.Join(lines, "\n"), false))
	} else if len(input.SampleMentions) > 0 {
		// Back-compat: producers that have not yet started emitting
		// sample_references still get rendered as plain quoted text.
		count := 3
		if len(input.SampleMentions) < 3 {
			count = len(input.SampleMentions)
		}
		mentions := input.SampleMentions[:count]
		quotedMentions := make([]string, len(mentions))
		for i, m := range mentions {
			quotedMentions[i] = fmt.Sprintf("> %s", m)
		}
		fields = append(fields, buildField("Sample Mentions", strings.Join(quotedMentions, "\n"), false))
	}

	// Determine MessageType based on severity
	msgType := discord.MessageTypeInfo
	switch strings.ToLower(input.Severity) {
	case "critical":
		msgType = discord.MessageTypeError
	case "warning":
		msgType = discord.MessageTypeWarning
	case "info":
		msgType = discord.MessageTypeInfo
	default:
		msgType = discord.MessageTypeError // Default to error if unknown high severity or fallback
	}

	opts := discord.MessageOptions{
		Type:        msgType,
		Title:       fmt.Sprintf("🚨 Crisis Alert: %s", input.ProjectName),
		Description: fmt.Sprintf("Unusual activity detected in project **%s** (%s).", input.ProjectName, input.ProjectID),
		Fields:      fields,
		Timestamp:   time.Now(),
		Footer: &discord.EmbedFooter{
			Text: "Notification Service • Crisis Monitor",
		},
	}

	return uc.discord.SendEmbed(ctx, opts)
}
