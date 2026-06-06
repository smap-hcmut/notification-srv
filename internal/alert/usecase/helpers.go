package usecase

import (
	"fmt"

	"github.com/smap-hcmut/shared-libs/go/discord"
)

func buildField(name string, value string, inline bool) discord.EmbedField {
	if value == "" {
		value = "N/A"
	}
	// Safety truncate for Discord field value limit (1024)
	if len(value) > 1024 {
		value = truncateText(value, 1024)
	}
	return discord.EmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max < 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
