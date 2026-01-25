# Align WebSocket Service with Event-Driven Architecture

**Status**: Draft  
**Created**: 2025-12-07  
**Change ID**: `align-websocket-event-driven`

---

## Why

The event-driven architecture specification (`document/event-drivent.md`) defines WebSocket Service as the final delivery layer for real-time notifications. However, gap analysis (`document/websocket_service_gap_analysis.md`) identified that the current implementation is missing critical message types (`project_progress`, `project_completed`) required for project execution progress notifications. Without these changes, clients cannot receive real-time updates during project crawling and analytics processing.

---

## What Changes

This proposal aligns the WebSocket Service with the event-driven architecture specification defined in `document/event-drivent.md`. The gap analysis in `document/websocket_service_gap_analysis.md` identified 7 gaps. This proposal addresses the **High Priority** gaps that are essential for the service to properly support project progress notifications.

### Priority Items (This Proposal)

| Gap                 | Description                                          | Effort |
| ------------------- | ---------------------------------------------------- | ------ |
| Message Types       | Add `project_progress` and `project_completed` types | Low    |
| Progress Payload    | Add typed struct with validation                     | Low    |
| Disconnect Callback | Integrate Hub → Subscriber notification              | Low    |
| Subscriber Health   | Add subscriber status to health endpoint             | Low    |

### Deferred Items (Future Work)

| Gap                        | Description                                    | Reason for Deferral          |
| -------------------------- | ---------------------------------------------- | ---------------------------- |
| Prometheus Metrics by Type | Add `websocket_messages_by_type_total` counter | Medium effort, non-blocking  |
| Redis Message Validation   | Warn on unknown message types                  | Low impact enhancement       |
| Circuit Breaker            | Add resilience for Redis failures              | Complex, needs design review |

---

## User Review Required

> [!IMPORTANT]
> The new message types (`project_progress`, `project_completed`) must match the schema published by Project Service. See `event-drivent.md` lines 676-706 for expected format.

---

## Proposed Changes

### 1. WebSocket Messages Component

#### [MODIFY] [message.go](file:///Users/tantai/Workspaces/smap/smap-api/websocket/internal/websocket/message.go)

Add new message types and payload structs:

```go
const (
    MessageTypeProjectProgress  MessageType = "project_progress"
    MessageTypeProjectCompleted MessageType = "project_completed"
)

// ProgressPayload represents project progress notification from Project Service
type ProgressPayload struct {
    ProjectID       string  `json:"project_id"`
    Status          string  `json:"status"`
    Total           int     `json:"total"`
    Done            int     `json:"done"`
    Errors          int     `json:"errors"`
    ProgressPercent float64 `json:"progress_percent"`
}
```

---

### 2. Hub-Subscriber Integration

#### [MODIFY] [hub.go](file:///Users/tantai/Workspaces/smap/smap-api/websocket/internal/websocket/hub.go)

Add `RedisNotifier` interface field and call on disconnect:

```go
type Hub struct {
    // ... existing fields ...
    redisNotifier RedisNotifier  // Optional callback for user disconnect
}

type RedisNotifier interface {
    OnUserDisconnected(userID string, hasOtherConnections bool) error
}
```

Update `unregisterConnection()` to call notifier.

---

### 3. Health Check Enhancement

#### [MODIFY] [health.go](file:///Users/tantai/Workspaces/smap/smap-api/websocket/internal/server/health.go)

Add subscriber health status:

```go
type HealthResponse struct {
    // ... existing fields ...
    Subscriber *SubscriberHealth `json:"subscriber"`
}

type SubscriberHealth struct {
    Active        bool      `json:"active"`
    LastMessageAt time.Time `json:"last_message_at,omitempty"`
    Pattern       string    `json:"pattern"`
}
```

---

### 4. Subscriber Observability

#### [MODIFY] [subscriber.go](file:///Users/tantai/Workspaces/smap/smap-api/websocket/internal/redis/subscriber.go)

Add tracking for last message timestamp and expose health method:

```go
type Subscriber struct {
    // ... existing fields ...
    lastMessageAt time.Time
    isActive      atomic.Bool
}

func (s *Subscriber) GetHealth() SubscriberHealth { ... }
```

---

## Verification Plan

### Automated Tests

**Unit Tests** (new file: `message_test.go`):

```bash
go test -v ./internal/websocket/... -run TestProgressPayload
go test -v ./internal/websocket/... -run TestMessageTypes
```

**Existing Tests** (verify no regression):

```bash
go test -v ./...
```

### Manual Verification

1. **Start service**:

   ```bash
   make run
   ```

2. **Verify health endpoint** includes subscriber status:

   ```bash
   curl http://localhost:8081/health | jq '.subscriber'
   ```

   Expected: `{"active": true, "pattern": "user_noti:*"}`

3. **Verify message type constants** are exported:
   ```go
   // In any consuming code
   ws.MessageTypeProjectProgress == "project_progress"
   ```
