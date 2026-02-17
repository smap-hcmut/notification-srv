package usecase

import (
	"fmt"
	"strings"

	"notification-srv/pkg/discord"
)

// mapSeverityToColor maps alert severity to Discord embed color.
func mapSeverityToColor(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 0xFF0000 // Red
	case "warning":
		return 0xFFA500 // Orange
	case "info":
		return 0x3498DB // Blue
	default:
		return 0x95A5A6 // Gray
	}
}

// mapStatusToColor maps onboarding status to Discord embed color.
func mapStatusToColor(status string) int {
	switch strings.ToLower(status) {
	case "completed":
		return 0x2ECC71 // Green
	case "failed":
		return 0xE74C3C // Red
	default:
		return 0x3498DB // Blue
	}
}

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
