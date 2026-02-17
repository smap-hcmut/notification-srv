# Alert Domain Code Plan

**Ref:** `documents/master-proposal.md`, `documents/domain_convention/convention.md`
**Status:** DRAFT
**Domain:** `internal/alert`

This document details the implementation plan for the `alert` domain (Domain #2), responsible for dispatching notifications to external channels like Discord.

---

## 1. Directory Structure

```
internal/alert/
├── interface.go                   # Public UseCase interface
├── types.go                       # Public Input/Output structs
├── errors.go                      # Domain sentinel errors
└── usecase/
    ├── new.go                     # Factory: NewUseCase
    ├── dispatch_crisis.go         # Logic: Crisis Alert -> Discord
    ├── dispatch_onboarding.go     # Logic: Data Onboarding -> Discord
    ├── dispatch_campaign.go       # Logic: Campaign Event -> Discord
    └── helpers.go                 # Helpers: Embed builders, formatting
```

---

## 2. Public Interfaces & Types (`root`)

### 2.1 `internal/alert/interface.go`

```go
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
```

### 2.2 `internal/alert/types.go`

```go
package alert

import "time"

// CrisisAlertInput represents a critical system or business alert.
type CrisisAlertInput struct {
 ProjectID       string
 ProjectName     string
 Severity        string   // e.g. "critical", "warning", "info"
 AlertType       string   // e.g. "spike", "drop", "sentiment"
 Metric          string   // e.g. "mention_count"
 CurrentValue    float64
 Threshold       float64
 AffectedAspects []string
 SampleMentions  []string // List of texts
 TimeWindow      string
 ActionRequired  string
 GeneratedAt     time.Time
}

// DataOnboardingInput represents a status update for a data source onboarding.
type DataOnboardingInput struct {
 ProjectID   string
 SourceID    string
 SourceName  string
 SourceType  string // e.g. "facebook_page", "instagram_account"
 Status      string // "completed", "failed"
 RecordCount int
 ErrorCount  int
 Message     string // Error details or success summary
 Duration    time.Duration
}

// CampaignEventInput represents a notification about a campaign state change.
type CampaignEventInput struct {
 CampaignID   string
 CampaignName string
 EventType    string // "created", "started", "paused", "finished"
 ResourceName string // e.g. "Keyword List", "Competitor List"
 ResourceURL  string
 User         string // Who triggered the event
 Message      string
 Timestamp    time.Time
}
```

### 2.3 `internal/alert/errors.go`

```go
package alert

import "errors"

var (
 ErrDispatchFailed = errors.New("failed to dispatch alert")
 ErrInvalidInput   = errors.New("invalid alert input")
)
```

---

## 3. UseCase Implementation (`usecase/`)

### 3.1 `usecase/new.go`

```go
package usecase

import (
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
```

### 3.2 `usecase/dispatch_crisis.go`

- **Logic**:
  - Validates input.
  - Maps `Severity` to Discord color (Red for Critical, Orange for Warning).
  - Builds a rich Embed with fields:
    - **Metric**: Current vs Threshold.
    - **Aspects**: Joined string.
    - **Mentions**: Quoted list.
    - **Action**: Bold text.
  - Calls `uc.discord.SendEmbed`.

### 3.3 `usecase/dispatch_onboarding.go`

- **Logic**:
  - Check Status: Only dispatch if `COMPLETED` or `FAILED` (ignore strictly progress updates to avoid spam, or as per requirement).
  - Color: Green (Success), Red (Failed).
  - Fields: Record Count, Error Count, Duration.

### 3.4 `usecase/dispatch_campaign.go`

- **Logic**:
  - Color: Blue (Info).
  - Fields: Event Type, Resource Link, User.

### 3.5 `usecase/helpers.go`

- **Responsibilities**:
  - `mapSeverityToColor(severity string) int`
  - `mapStatusToColor(status string) int`
  - `formatFloat(f float64) string`
  - `truncateText(s string, max int) string`

---

## 4. Implementation Checklist

- [ ] **Infrastructure**:
  - [ ] `internal/alert/interface.go` (Update stub)
  - [ ] `internal/alert/types.go`
  - [ ] `internal/alert/errors.go`

- [ ] **UseCase Logic**:
  - [ ] `internal/alert/usecase/new.go` (Update stub)
  - [ ] `internal/alert/usecase/helpers.go`
  - [ ] `internal/alert/usecase/dispatch_crisis.go`
  - [ ] `internal/alert/usecase/dispatch_onboarding.go`
  - [ ] `internal/alert/usecase/dispatch_campaign.go`

- [ ] **Wiring**:
  - [ ] Already wired in `internal/httpserver/handler.go` (Verified).
  - [ ] Verify `pkg/discord` injection is working correctness.

---
