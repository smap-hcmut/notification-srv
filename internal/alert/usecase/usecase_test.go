package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"notification-srv/internal/alert"

	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

type noopLogger struct{}

func (noopLogger) Debug(context.Context, ...any)           {}
func (noopLogger) Debugf(context.Context, string, ...any)  {}
func (noopLogger) Info(context.Context, ...any)            {}
func (noopLogger) Infof(context.Context, string, ...any)   {}
func (noopLogger) Warn(context.Context, ...any)            {}
func (noopLogger) Warnf(context.Context, string, ...any)   {}
func (noopLogger) Error(context.Context, ...any)           {}
func (noopLogger) Errorf(context.Context, string, ...any)  {}
func (noopLogger) DPanic(context.Context, ...any)          {}
func (noopLogger) DPanicf(context.Context, string, ...any) {}
func (noopLogger) Panic(context.Context, ...any)           {}
func (noopLogger) Panicf(context.Context, string, ...any)  {}
func (noopLogger) Fatal(context.Context, ...any)           {}
func (noopLogger) Fatalf(context.Context, string, ...any)  {}
func (l noopLogger) WithTrace(context.Context) log.Logger  { return l }

type fakeDiscord struct {
	sendEmbedErr error
	embeds       []discord.MessageOptions
}

func (f *fakeDiscord) SendMessage(context.Context, string) error { return nil }
func (f *fakeDiscord) SendEmbed(_ context.Context, options discord.MessageOptions) error {
	f.embeds = append(f.embeds, options)
	return f.sendEmbedErr
}
func (f *fakeDiscord) SendError(context.Context, string, string, error) error { return nil }
func (f *fakeDiscord) SendSuccess(context.Context, string, string) error      { return nil }
func (f *fakeDiscord) SendWarning(context.Context, string, string) error      { return nil }
func (f *fakeDiscord) SendInfo(context.Context, string, string) error         { return nil }
func (f *fakeDiscord) ReportBug(context.Context, string) error                { return nil }
func (f *fakeDiscord) SendNotification(context.Context, string, string, map[string]string) error {
	return nil
}
func (f *fakeDiscord) SendActivityLog(context.Context, string, string, string) error { return nil }
func (f *fakeDiscord) GetWebhookURL() string                                         { return "" }
func (f *fakeDiscord) Close() error                                                  { return nil }

func TestNew(t *testing.T) {
	discordClient := &fakeDiscord{}
	uc := New(noopLogger{}, discordClient)
	impl, ok := uc.(*implUseCase)
	require.True(t, ok)
	require.Equal(t, discordClient, impl.discord)
}

func TestMapSeverityToColor(t *testing.T) {
	tcs := map[string]struct {
		input string
		want  int
	}{
		"critical":      {"critical", 0xFF0000},
		"warning upper": {"WARNING", 0xFFA500},
		"info":          {"info", 0x3498DB},
		"default":       {"unknown", 0x95A5A6},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, mapSeverityToColor(tc.input))
		})
	}
}

func TestMapStatusToColor(t *testing.T) {
	tcs := map[string]struct {
		input string
		want  int
	}{
		"completed":    {"completed", 0x2ECC71},
		"failed upper": {"FAILED", 0xE74C3C},
		"default":      {"running", 0x3498DB},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, mapStatusToColor(tc.input))
		})
	}
}

func TestBuildField(t *testing.T) {
	tcs := map[string]struct {
		name   string
		value  string
		inline bool
		want   discord.EmbedField
	}{
		"empty value": {
			name:   "Field",
			value:  "",
			inline: true,
			want:   discord.EmbedField{Name: "Field", Value: "N/A", Inline: true},
		},
		"truncated": {
			name:   "Field",
			value:  strings.Repeat("a", 1030),
			inline: false,
			want:   discord.EmbedField{Name: "Field", Value: strings.Repeat("a", 1021) + "..."},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, buildField(tc.name, tc.value, tc.inline))
		})
	}
}

func TestFormatFloat(t *testing.T) {
	require.Equal(t, "12.35", formatFloat(12.345))
}

func TestTruncateText(t *testing.T) {
	tcs := map[string]struct {
		input string
		max   int
		want  string
	}{
		"short":         {"abc", 10, "abc"},
		"tiny max":      {"abcdef", 2, "ab"},
		"with ellipsis": {"abcdef", 5, "ab..."},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, truncateText(tc.input, tc.max))
		})
	}
}

func TestDispatchCrisisAlert(t *testing.T) {
	expectedErr := errors.New("discord error")
	tcs := map[string]struct {
		input       alert.CrisisAlertInput
		err         error
		wantType    discord.MessageType
		wantFields  []string
		wantSamples int
	}{
		"critical with details": {
			input: alert.CrisisAlertInput{
				ProjectID:       "project-1",
				ProjectName:     "Launch",
				Severity:        "critical",
				AlertType:       "spike",
				Metric:          "mention_count",
				CurrentValue:    125.456,
				Threshold:       99.1,
				AffectedAspects: []string{"price", "quality"},
				SampleMentions:  []string{"one", "two", "three", "four"},
				TimeWindow:      "10m",
				ActionRequired:  "Investigate",
			},
			wantType:    discord.MessageTypeError,
			wantFields:  []string{"Severity", "Alert Type", "Metric", "Value vs Threshold", "Time Window", "Action Required", "Affected Aspects", "Sample Mentions"},
			wantSamples: 3,
		},
		"warning propagates discord error": {
			input: alert.CrisisAlertInput{
				ProjectID:      "project-1",
				ProjectName:    "Launch",
				Severity:       "warning",
				AlertType:      "drop",
				Metric:         "sentiment",
				TimeWindow:     "1h",
				SampleMentions: []string{"one"},
			},
			err:      expectedErr,
			wantType: discord.MessageTypeWarning,
		},
		"info severity": {
			input:    alert.CrisisAlertInput{ProjectID: "p", ProjectName: "P", Severity: "info"},
			wantType: discord.MessageTypeInfo,
		},
		"unknown severity defaults to error": {
			input:    alert.CrisisAlertInput{ProjectID: "p", ProjectName: "P", Severity: "bad"},
			wantType: discord.MessageTypeError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			discordClient := &fakeDiscord{sendEmbedErr: tc.err}
			uc := New(noopLogger{}, discordClient)

			err := uc.DispatchCrisisAlert(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			require.Len(t, discordClient.embeds, 1)
			got := discordClient.embeds[0]
			require.Equal(t, tc.wantType, got.Type)
			require.Contains(t, got.Title, tc.input.ProjectName)
			require.NotZero(t, got.Timestamp)
			require.NotNil(t, got.Footer)

			if len(tc.wantFields) > 0 {
				require.Len(t, got.Fields, len(tc.wantFields))
				for i, field := range tc.wantFields {
					require.Equal(t, field, got.Fields[i].Name)
				}
				require.Equal(t, tc.wantSamples, strings.Count(got.Fields[len(got.Fields)-1].Value, "> "))
			}
		})
	}
}

func TestDispatchDataOnboarding(t *testing.T) {
	expectedErr := errors.New("discord error")
	tcs := map[string]struct {
		input     alert.DataOnboardingInput
		err       error
		wantCalls int
		wantType  discord.MessageType
		wantTitle string
	}{
		"completed": {
			input: alert.DataOnboardingInput{
				ProjectID: "project-1", SourceName: "Facebook", SourceType: "facebook_page",
				Status: "completed", RecordCount: 10, ErrorCount: 0, Duration: time.Minute, Message: "done",
			},
			wantCalls: 1,
			wantType:  discord.MessageTypeSuccess,
			wantTitle: "Data Onboarding: Completed",
		},
		"failed propagates discord error": {
			input: alert.DataOnboardingInput{
				ProjectID: "project-1", SourceName: "Instagram", SourceType: "instagram_account",
				Status: "FAILED", RecordCount: 10, ErrorCount: 2,
			},
			err:       expectedErr,
			wantCalls: 1,
			wantType:  discord.MessageTypeError,
			wantTitle: "Data Onboarding FAILED: Instagram",
		},
		"running skipped": {
			input:     alert.DataOnboardingInput{Status: "running"},
			wantCalls: 0,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			discordClient := &fakeDiscord{sendEmbedErr: tc.err}
			uc := New(noopLogger{}, discordClient)

			err := uc.DispatchDataOnboarding(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			require.Len(t, discordClient.embeds, tc.wantCalls)
			if tc.wantCalls == 0 {
				return
			}
			got := discordClient.embeds[0]
			require.Equal(t, tc.wantType, got.Type)
			require.Equal(t, tc.wantTitle, got.Title)
			require.NotZero(t, got.Timestamp)
			require.NotNil(t, got.Footer)
		})
	}
}

func TestDispatchCampaignEvent(t *testing.T) {
	expectedErr := errors.New("discord error")
	tcs := map[string]struct {
		input        alert.CampaignEventInput
		err          error
		wantFields   int
		wantResource string
	}{
		"with resource link and message": {
			input: alert.CampaignEventInput{
				CampaignID: "campaign-1", CampaignName: "Spring", EventType: "started", User: "admin",
				ResourceName: "Keyword List", ResourceURL: "https://example.test/list", Message: "ready",
			},
			wantFields:   5,
			wantResource: "[Keyword List](https://example.test/list)",
		},
		"without optional fields propagates error": {
			input:      alert.CampaignEventInput{CampaignID: "campaign-1", CampaignName: "Spring", EventType: "paused", User: "admin"},
			err:        expectedErr,
			wantFields: 3,
		},
		"resource without url": {
			input:        alert.CampaignEventInput{CampaignID: "campaign-1", CampaignName: "Spring", ResourceName: "Keyword List"},
			wantFields:   4,
			wantResource: "Keyword List",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			discordClient := &fakeDiscord{sendEmbedErr: tc.err}
			uc := New(noopLogger{}, discordClient)

			err := uc.DispatchCampaignEvent(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.err)
			require.Len(t, discordClient.embeds, 1)
			got := discordClient.embeds[0]
			require.Equal(t, discord.MessageTypeInfo, got.Type)
			require.Len(t, got.Fields, tc.wantFields)
			if tc.wantResource != "" {
				require.Equal(t, tc.wantResource, got.Fields[3].Value)
			}
			require.NotZero(t, got.Timestamp)
			require.NotNil(t, got.Footer)
		})
	}
}
