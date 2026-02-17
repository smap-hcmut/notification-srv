# WebSocket Domain Refactor — Code Plan

**Ref:** `documents/master-proposal.md`, `documents/domain_convention/`  
**Status:** PLANNING  
**Last Updated:** 17/02/2026

---

## 1. Vấn đề hiện tại

### 1.1 Cấu trúc domain `websocket` hiện tại

```
internal/websocket/
├── types.go                    # Message/BroadcastMessage + 10 MessageType constants (legacy)
├── errors.go                   # 6 sentinel errors
├── delivery/
│   ├── http/
│   │   ├── handlers.go         # 395 dòng - MIX: upgrader, CORS check, cookie, auth, handler, WS config
│   │   ├── topic_validation.go # Topic parsing logic
│   │   └── topic_validation_test.go
│   └── redis/
│       └── subscriber.go       # IMPORT trực tiếp transform/usecase → TIGHT COUPLING
├── usecase/
│   ├── hub.go                  # Hub struct + Run() + Register/Unregister + Send methods
│   ├── connection.go           # Connection struct + readPump/writePump
│   ├── hub_test.go
│   ├── connection_test.go
│   ├── load_test.go
│   └── chaos_test.go
├── integration_test.go
├── compatibility_test.go
├── mock_rate_limiter_test.go
└── types_test.go
```

### 1.2 Cấu trúc domain `transform` hiện tại

```
internal/transform/
├── interface.go    # 4 interfaces: MessageTransformer, MessageValidator, MetricsCollector, ErrorHandler
├── types.go        # ~600 dòng - BLOATED: Project/Job Input+Output + Batch + Content + Author + Metrics + Media
├── enums.go        # Legacy enums: ProjectStatus, JobStatus, Platform (TikTok/YouTube/Instagram)
├── errors.go       # Error constructors
├── constant.go     # MaxLogLength, MaxLatencySize, MaxIDLength
├── util.go         # ValidateTopicFormat, splitTopic
└── usecase/
    ├── new.go      # 6 factory functions (transformer, validator, metrics, errorHandler, project, job)
    ├── types.go    # 6 impl structs (inputValidator, errorHandler, metricsCollector, transformer, project, job)
    ├── transform.go  # TransformMessage dispatcher
    ├── validator.go  # Input validation
    ├── project.go    # Project-specific transform
    ├── job.go        # Job-specific transform ← LEGACY, DELETE
    ├── metrics.go    # Transform metrics
    └── error_handler.go # Error handling
```

### 1.3 Các vấn đề cụ thể

| #   | Vấn đề                                                                                                               | Severity |
| --- | -------------------------------------------------------------------------------------------------------------------- | -------- |
| 1   | `delivery/http/handlers.go` (395 dòng) mix quá nhiều concern: CORS, upgrader, cookie, auth, WS config, handler logic | HIGH     |
| 2   | `delivery/redis/subscriber.go` import trực tiếp `transform/usecase` → vi phạm Clean Architecture                     | HIGH     |
| 3   | `transform/types.go` chứa legacy types: Job, Batch, Content, Author, Metrics, Media                                  | HIGH     |
| 4   | `transform/enums.go` chứa legacy enums: JobStatus, Platform (TikTok/YouTube/Instagram)                               | HIGH     |
| 5   | Không có `interface.go` ở module root `websocket/` cho UseCase                                                       | HIGH     |
| 6   | WebSocket `types.go` có legacy MessageType: `MessageTypeJobProgress`, `MessageTypeJobCompleted`                      | MEDIUM   |
| 7   | Transform có 4 interfaces quá phức tạp (Transformer, Validator, Metrics, ErrorHandler)                               | MEDIUM   |
| 8   | Subscriber vừa parse channel, vừa transform, vừa route → God Object                                                  | HIGH     |
| 9   | Không có Alert Dispatcher (Discord integration cho crisis alerts)                                                    | HIGH     |
| 10  | Thiếu message types mới: DATA_ONBOARDING, ANALYTICS_PIPELINE, CRISIS_ALERT, CAMPAIGN_EVENT                           | HIGH     |

---

## 2. Thiết kế mới

### 2.1 Domain boundaries

```
Trước:
  websocket domain  ←→  transform domain  (tight coupling)

Sau:
  notification domain (gộp websocket + transform + alert)
  ├── websocket   = connection management (Hub, Connection, read/writePump)
  ├── transform   = message validation & transformation
  └── alert       = crisis alert dispatching (Discord)
```

**Quyết định kiến trúc:**

- **Gộp `transform` vào `websocket`** vì transform chỉ phục vụ websocket, không có consumer nào khác.
- **Tạo domain mới `alert`** cho Discord dispatching (tách biệt concern).
- Redis subscriber **chỉ là delivery** (nhận message, parse channel, gọi usecase).
- Transform logic **là usecase** (business logic: validate input → transform → validate output).

### 2.2 Cấu trúc mới (theo convention)

```
internal/
├── websocket/                         # WebSocket notification domain
│   ├── interface.go                   # UseCase interface definition
│   ├── types.go                       # ALL Input/Output structs (public)
│   ├── errors.go                      # Module-specific sentinel errors
│   │
│   ├── usecase/
│   │   ├── new.go                     # implUseCase struct + New() factory
│   │   ├── hub.go                     # Hub: Run(), Register, Unregister, broadcast logic
│   │   ├── connection.go              # Connection: readPump, writePump, lifecycle
│   │   ├── transform.go              # TransformMessage(): validate → transform → route
│   │   ├── helpers.go                 # Private helpers (clamp, parse, validate format)
│   │   └── types.go                   # Private structs (internal to usecase)
│   │
│   ├── delivery/
│   │   ├── http/
│   │   │   ├── new.go                 # Handler interface + New() factory
│   │   │   ├── handlers.go            # HandleWebSocket (thin: auth → upgrade → register)
│   │   │   ├── process_request.go     # processUpgradeReq: extract token, projectID, filters
│   │   │   ├── presenters.go          # UpgradeReq DTO, toInput()
│   │   │   ├── routes.go              # RegisterRoutes(rg, mw)
│   │   │   └── errors.go             # mapError: domain err → HTTP err
│   │   │
│   │   └── redis/
│   │       ├── new.go                 # Subscriber interface + New() factory
│   │       ├── subscriber.go          # Start(), listen(), reconnect(), Shutdown()
│   │       ├── workers.go             # handleMessage(): parse channel → uc.ProcessMessage()
│   │       └── presenters.go          # RedisMessage DTO, channel parsing helpers
│   │
│   └── tests/                         # Giữ lại integration/compatibility tests
│       ├── integration_test.go
│       └── compatibility_test.go
│
├── alert/                             # Alert dispatching domain (NEW)
│   ├── interface.go                   # UseCase interface: DispatchCrisisAlert, DispatchOnboarding, etc.
│   ├── types.go                       # CrisisAlertInput, CampaignEventInput, etc.
│   ├── errors.go                      # Alert-specific errors
│   │
│   └── usecase/
│       ├── new.go                     # implUseCase + New(discord, logger)
│       ├── dispatch_crisis.go         # DispatchCrisisAlert() → Discord webhook
│       ├── dispatch_onboarding.go     # DispatchDataOnboarding() → Discord (COMPLETED/FAILED only)
│       ├── dispatch_campaign.go       # DispatchCampaignEvent() → Discord
│       └── helpers.go                 # Discord embed builders, severity color mapping
│
├── model/                             # Shared domain models
│   ├── constant.go                    # Environment enum
│   └── scope.go                       # Scope struct
│
├── httpserver/                        # HTTP server lifecycle (existing, keep)
│   ├── new.go
│   ├── httpserver.go
│   ├── handler.go
│   └── health.go
│
└── middleware/                        # Middleware (existing, keep)
    ├── new.go
    ├── cors.go
    ├── recovery.go
    └── auth.go
```

---

## 3. Chi tiết từng file

### 3.1 `internal/websocket/interface.go` (NEW)

```go
package websocket

import "context"

// UseCase defines the business logic interface for the WebSocket notification domain.
// Implementations are safe for concurrent use.
type UseCase interface {
    // Hub lifecycle
    Run()
    Shutdown(ctx context.Context) error

    // Connection management
    Register(conn Connection)
    Unregister(conn Connection)
    GetStats() HubStats

    // Message processing (called by Redis subscriber delivery)
    ProcessMessage(ctx context.Context, input ProcessMessageInput) error

    // Notifier callbacks (called by Redis subscriber for Pub/Sub lifecycle)
    OnUserConnected(userID string) error
    OnUserDisconnected(userID string, hasOtherConnections bool) error
}
```

**Giải thích:** Một flat interface, gộp Hub + Transform logic. Redis subscriber (delivery) chỉ cần gọi `ProcessMessage()` — tất cả business logic (validate, transform, route, dispatch alert) nằm trong usecase.

---

### 3.2 `internal/websocket/types.go` (REWRITE)

```go
package websocket

import "time"

// --- Message Types (new analytics platform) ---

type MessageType string

const (
    MessageTypeDataOnboarding    MessageType = "DATA_ONBOARDING"
    MessageTypeAnalyticsPipeline MessageType = "ANALYTICS_PIPELINE"
    MessageTypeCrisisAlert       MessageType = "CRISIS_ALERT"
    MessageTypeCampaignEvent     MessageType = "CAMPAIGN_EVENT"
    MessageTypeSystem            MessageType = "SYSTEM"
)

// --- Channel Types ---

type ChannelType string

const (
    ChannelTypeProject  ChannelType = "project"
    ChannelTypeCampaign ChannelType = "campaign"
    ChannelTypeAlert    ChannelType = "alert"
    ChannelTypeSystem   ChannelType = "system"
)

// --- UseCase Input/Output DTOs ---

// ProcessMessageInput is the input for UseCase.ProcessMessage().
// This is what the Redis subscriber delivery passes to the usecase.
type ProcessMessageInput struct {
    Channel string // Raw Redis channel (e.g. "project:proj_vf8:user:user123")
    Payload []byte // Raw JSON payload from Redis
}

// ParsedChannel is the result of parsing a Redis channel string.
type ParsedChannel struct {
    ChannelType ChannelType
    EntityID    string // project_id, campaign_id, etc.
    UserID      string // Target user (empty for broadcast channels like system:*)
    SubType     string // For alert channels: "crisis", "warning"
}

// HubStats contains Hub runtime statistics (for health checks).
type HubStats struct {
    ActiveConnections int
    TotalUniqueUsers  int
}

// --- Data Onboarding ---

type DataOnboardingPayload struct {
    ProjectID   string `json:"project_id"`
    SourceID    string `json:"source_id"`
    SourceName  string `json:"source_name"`
    SourceType  string `json:"source_type"`
    Status      string `json:"status"`
    Progress    int    `json:"progress"`
    RecordCount int    `json:"record_count"`
    ErrorCount  int    `json:"error_count"`
    Message     string `json:"message"`
}

// --- Analytics Pipeline ---

type AnalyticsPipelinePayload struct {
    ProjectID       string `json:"project_id"`
    SourceID        string `json:"source_id"`
    TotalRecords    int    `json:"total_records"`
    ProcessedCount  int    `json:"processed_count"`
    SuccessCount    int    `json:"success_count"`
    FailedCount     int    `json:"failed_count"`
    Progress        int    `json:"progress"`
    CurrentPhase    string `json:"current_phase"`
    EstimatedTimeMs int64  `json:"estimated_time_ms"`
}

// --- Crisis Alert ---

type CrisisAlertPayload struct {
    ProjectID       string   `json:"project_id"`
    ProjectName     string   `json:"project_name"`
    Severity        string   `json:"severity"`
    AlertType       string   `json:"alert_type"`
    Metric          string   `json:"metric"`
    CurrentValue    float64  `json:"current_value"`
    Threshold       float64  `json:"threshold"`
    AffectedAspects []string `json:"affected_aspects"`
    SampleMentions  []string `json:"sample_mentions"`
    TimeWindow      string   `json:"time_window"`
    ActionRequired  string   `json:"action_required"`
}

// --- Campaign Event ---

type CampaignEventPayload struct {
    CampaignID   string `json:"campaign_id"`
    CampaignName string `json:"campaign_name"`
    EventType    string `json:"event_type"`
    ResourceID   string `json:"resource_id"`
    ResourceName string `json:"resource_name"`
    ResourceURL  string `json:"resource_url"`
    Message      string `json:"message"`
}

// --- Notification Output (WebSocket → Browser) ---

type NotificationOutput struct {
    Type      MessageType `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
    Payload   interface{} `json:"payload"`
}

// --- Connection DTO (for UseCase interface) ---

type ConnectionInput struct {
    UserID    string
    ProjectID string // Optional filter
    Conn      interface{} // *websocket.Conn (opaque to avoid importing gorilla in interface)
}
```

**Lưu ý:**

- Xóa hoàn toàn: `MessageTypeJobProgress`, `MessageTypeJobCompleted`, `MessageTypeDryRunResult`
- Xóa: `ProgressPayload`, `BroadcastMessage` (sẽ define lại nếu cần trong `usecase/types.go`)
- Tất cả payload struct theo đúng master-proposal.md

---

### 3.3 `internal/websocket/errors.go` (UPDATE)

```go
package websocket

import "errors"

// Connection errors
var (
    ErrInvalidToken          = errors.New("invalid or expired JWT token")
    ErrMissingToken          = errors.New("missing JWT token")
    ErrConnectionClosed      = errors.New("connection closed")
    ErrMaxConnectionsReached = errors.New("maximum connections reached")
    ErrUserNotFound          = errors.New("user not found in connection registry")
)

// Message errors
var (
    ErrInvalidMessage     = errors.New("invalid message format")
    ErrUnknownMessageType = errors.New("unknown message type")
    ErrInvalidChannel     = errors.New("invalid Redis channel format")
)

// Transform errors
var (
    ErrTransformFailed = errors.New("message transformation failed")
    ErrValidationFailed = errors.New("message validation failed")
)
```

---

### 3.4 `internal/websocket/usecase/new.go` (REWRITE)

```go
package usecase

import (
    "notification-srv/internal/websocket"
    "notification-srv/internal/alert"
    "notification-srv/pkg/log"
)

// implUseCase implements websocket.UseCase.
type implUseCase struct {
    hub            *Hub
    logger         log.Logger
    alertUC        alert.UseCase     // For dispatching crisis alerts to Discord
    maxConnections int
}

// New creates a new WebSocket UseCase.
func New(logger log.Logger, maxConnections int, alertUC alert.UseCase) websocket.UseCase {
    hub := newHub(logger, maxConnections)
    return &implUseCase{
        hub:            hub,
        logger:         logger,
        alertUC:        alertUC,
        maxConnections: maxConnections,
    }
}
```

**Thay đổi chính:**

- Hub trở thành **private** (`newHub` thay vì `NewHub`), chỉ expose qua UseCase interface
- Alert UseCase inject vào (cho crisis alert dispatching)
- Return **interface** `websocket.UseCase`, không return `*Hub`

---

### 3.5 `internal/websocket/usecase/hub.go` (REFACTOR)

Giữ nguyên core logic (connections map, register/unregister channels, Run loop). Thay đổi:

- Đổi `Hub` thành **unexported** nếu muốn strict, hoặc giữ exported cho test
- Remove `SetRedisNotifier()` (subscriber sẽ gọi UseCase interface trực tiếp)
- Remove `GetStats()` → move lên `implUseCase.GetStats()`

---

### 3.6 `internal/websocket/usecase/transform.go` (NEW — gộp từ transform domain)

```go
package usecase

import (
    "context"
    "encoding/json"
    "fmt"

    ws "notification-srv/internal/websocket"
)

// ProcessMessage validates, transforms, and routes a Redis message.
// This is the main business logic entry point called by Redis subscriber.
func (uc *implUseCase) ProcessMessage(ctx context.Context, input ws.ProcessMessageInput) error {
    // 1. Parse channel
    parsed, err := parseChannel(input.Channel)
    if err != nil {
        return fmt.Errorf("parse channel: %w", err)
    }

    // 2. Detect message type
    msgType, err := detectMessageType(input.Payload)
    if err != nil {
        return fmt.Errorf("detect type: %w", err)
    }

    // 3. Validate & Transform
    output, err := uc.transformMessage(ctx, msgType, input.Payload)
    if err != nil {
        return fmt.Errorf("transform: %w", err)
    }

    // 4. Dispatch to alert channel (Discord) if needed
    if msgType == ws.MessageTypeCrisisAlert {
        if alertErr := uc.dispatchAlert(ctx, msgType, input.Payload); alertErr != nil {
            uc.logger.Warnf(ctx, "alert dispatch failed (non-blocking): %v", alertErr)
        }
    }

    // 5. Route to WebSocket connections
    outputBytes, err := json.Marshal(output)
    if err != nil {
        return fmt.Errorf("marshal output: %w", err)
    }

    uc.routeMessage(parsed, outputBytes)
    return nil
}
```

---

### 3.7 `internal/websocket/delivery/redis/` (REFACTOR)

**Mục tiêu:** Subscriber trở thành **thin delivery layer** — chỉ nhận message từ Redis và gọi `uc.ProcessMessage()`.

#### `new.go`

```go
package redis

type Subscriber interface {
    Start() error
    Shutdown(ctx context.Context) error
    OnUserConnected(userID string) error
    OnUserDisconnected(userID string, hasOtherConnections bool) error
}

type subscriber struct {
    redis  pkgRedis.IRedis
    uc     websocket.UseCase
    logger log.Logger
}

func New(redis pkgRedis.IRedis, uc websocket.UseCase, logger log.Logger) Subscriber {
    return &subscriber{redis: redis, uc: uc, logger: logger}
}
```

#### `workers.go`

```go
func (s *subscriber) handleMessage(channel, payload string) {
    ctx := context.Background()

    input := websocket.ProcessMessageInput{
        Channel: channel,
        Payload: []byte(payload),
    }

    if err := s.uc.ProcessMessage(ctx, input); err != nil {
        s.logger.Warnf(ctx, "process message failed: channel=%s err=%v", channel, err)
    }
}
```

**Thay đổi chính:**

- Subscriber **KHÔNG import** `transform` package → gọi `uc.ProcessMessage()` thay vì tự transform
- Subscriber **nhận** `websocket.UseCase` interface thay vì `*usecase.Hub`
- Channel parsing move ra `presenters.go` hoặc vào usecase (vì nó là business logic)

---

### 3.8 `internal/websocket/delivery/http/` (REFACTOR theo convention)

Tách `handlers.go` (395 dòng) thành 5 file:

| File cũ (1 file)     | File mới             | Nội dung                                |
| -------------------- | -------------------- | --------------------------------------- |
| handlers.go L1-50    | `new.go`             | Handler interface + New() factory       |
| handlers.go L50-150  | `process_request.go` | Token extraction, filter parsing        |
| handlers.go L150-250 | `handlers.go`        | HandleWebSocket (thin controller)       |
| handlers.go (types)  | `presenters.go`      | WSConfig, CookieConfig DTOs, UpgradeReq |
| handlers.go L250-395 | `routes.go`          | SetupRoutes()                           |
| (new)                | `errors.go`          | mapError() for WebSocket domain errors  |

---

### 3.9 `internal/alert/` (NEW domain)

#### `interface.go`

```go
package alert

import "context"

// UseCase defines the alert dispatching interface.
type UseCase interface {
    DispatchCrisisAlert(ctx context.Context, input CrisisAlertInput) error
    DispatchDataOnboarding(ctx context.Context, input DataOnboardingInput) error
    DispatchCampaignEvent(ctx context.Context, input CampaignEventInput) error
}
```

#### `types.go`

```go
package alert

type CrisisAlertInput struct {
    ProjectID       string
    ProjectName     string
    Severity        string
    AlertType       string
    Metric          string
    CurrentValue    float64
    Threshold       float64
    AffectedAspects []string
    SampleMentions  []string
    TimeWindow      string
    ActionRequired  string
}

type DataOnboardingInput struct {
    SourceName string
    SourceType string
    Status     string
    Progress   int
    RecordCount int
    ErrorCount  int
    Message    string
}

type CampaignEventInput struct {
    CampaignName string
    EventType    string
    ResourceName string
    ResourceURL  string
    Message      string
}
```

#### `usecase/new.go`

```go
package usecase

type implUseCase struct {
    discord discord.IDiscord
    logger  log.Logger
}

func New(logger log.Logger, discord discord.IDiscord) alert.UseCase {
    return &implUseCase{discord: discord, logger: logger}
}
```

#### `usecase/dispatch_crisis.go`

- Build Discord embed message
- Severity → color mapping
- Send via `discord.IDiscord`

#### `usecase/dispatch_onboarding.go`

- Only send for COMPLETED/FAILED status
- Build simple embed

#### `usecase/dispatch_campaign.go`

- Build campaign event embed

#### `usecase/helpers.go`

- `severityToColor()`, `buildEmbedFields()`, etc.

---

## 4. Xóa / Delete

### 4.1 Delete hoàn toàn `internal/transform/`

Toàn bộ domain `transform` sẽ bị xóa. Logic cần thiết sẽ được gộp vào `internal/websocket/usecase/`:

- `transform.go` (TransformMessage dispatcher) → `websocket/usecase/transform.go`
- `project.go` (project transform) → `websocket/usecase/helpers.go`
- `validator.go` → `websocket/usecase/helpers.go`
- `job.go` → **DELETE** (legacy)
- `metrics.go` → **DELETE** (over-engineering cho thesis)
- `error_handler.go` → **DELETE** (dùng standard error handling)
- `interface.go` (4 interfaces) → **DELETE** (simplify thành 1 UseCase method)
- `types.go` (legacy types) → **REWRITE** in `websocket/types.go`
- `enums.go` (legacy enums) → **REWRITE** new enums in `websocket/types.go`

### 4.2 Delete legacy types

```
❌ DELETE: JobInputMessage, JobNotificationMessage
❌ DELETE: BatchInput, BatchData
❌ DELETE: ContentInput, ContentItem, AuthorInput, AuthorInfo
❌ DELETE: MetricsInput, EngagementMetrics, MediaInput, MediaInfo
❌ DELETE: JobStatus, Platform (TikTok/YouTube/Instagram)
❌ DELETE: ChannelPatternUserNoti ("user_noti:*"), ChannelPatternJob ("job:*")
❌ DELETE: MessageTypeJobProgress, MessageTypeJobCompleted, MessageTypeDryRunResult
```

---

## 5. Dependency Graph (sau refactor)

```
cmd/api/main.go
    ├── config.Load()
    ├── pkg/log, pkg/redis, pkg/discord, pkg/scope
    ├── internal/alert/usecase.New(logger, discord) → alert.UseCase
    ├── internal/httpserver.New(logger, cfg)
    │   └── (trong Run/mapHandlers)
    │       ├── internal/websocket/usecase.New(logger, maxConn, alertUC) → websocket.UseCase
    │       ├── internal/websocket/delivery/redis.New(redis, wsUC, logger) → Subscriber
    │       ├── internal/websocket/delivery/http.New(wsUC, logger) → Handler
    │       └── internal/middleware.New(logger, scope, cookie)

Dependency flow (Clean Architecture):
    delivery/redis  → websocket.UseCase (interface)
    delivery/http   → websocket.UseCase (interface)
    usecase         → alert.UseCase (interface)
    usecase         → pkg/log (shared lib)
    alert/usecase   → pkg/discord (shared lib)
```

**Không có circular dependency.** Mỗi layer chỉ depend lên interface.

---

## 6. Wiring trong `internal/httpserver/handler.go`

```go
func (srv *HTTPServer) mapHandlers() error {
    mw := middleware.New(srv.logger, srv.jwtMgr, srv.cookieCfg)
    srv.registerMiddlewares(mw)
    srv.registerSystemRoutes()

    // 1. Init Alert UseCase
    alertUC := alertUsecase.New(srv.logger, srv.discord)

    // 2. Init WebSocket UseCase (includes Hub internally)
    wsUC := wsUsecase.New(srv.logger, srv.wsConfig.MaxConnections, alertUC)

    // 3. Init Redis Subscriber (delivery, calls wsUC.ProcessMessage)
    subscriber := redisDelivery.New(srv.redis, wsUC, srv.logger)
    if err := subscriber.Start(); err != nil {
        return err
    }

    // 4. Init WebSocket HTTP Handler (delivery, calls wsUC.Register)
    wsHandler := wsHTTPDelivery.New(wsUC, srv.jwtMgr, srv.logger, srv.wsConfig, srv.cookieCfg, srv.environment)
    wsHandler.RegisterRoutes(srv.gin, mw)

    // 5. Store references for lifecycle management
    srv.wsUC = wsUC
    srv.subscriber = subscriber

    return nil
}
```

---

## 7. Migration Checklist

### Phase 1: Tạo domain `alert` (tách Discord logic)

- [ ] Tạo `internal/alert/interface.go`
- [ ] Tạo `internal/alert/types.go`
- [ ] Tạo `internal/alert/errors.go`
- [ ] Tạo `internal/alert/usecase/new.go`
- [ ] Tạo `internal/alert/usecase/dispatch_crisis.go`
- [ ] Tạo `internal/alert/usecase/dispatch_onboarding.go`
- [ ] Tạo `internal/alert/usecase/dispatch_campaign.go`
- [ ] Tạo `internal/alert/usecase/helpers.go`

### Phase 2: Rewrite `websocket/types.go` và `errors.go`

- [ ] Xóa legacy types (Job, Batch, Content, Author, Metrics, Media)
- [ ] Thêm new types (DataOnboarding, AnalyticsPipeline, CrisisAlert, CampaignEvent)
- [ ] Thêm new channel types và parsing
- [ ] Thêm new MessageType constants
- [ ] Update errors.go

### Phase 3: Tạo `websocket/interface.go`

- [ ] Define UseCase interface (flat, grouped comments)
- [ ] ProcessMessage() as the main entry point

### Phase 4: Refactor `websocket/usecase/`

- [ ] Tạo `new.go` (factory, inject alert.UseCase)
- [ ] Refactor `hub.go` (giữ core, remove SetRedisNotifier)
- [ ] Giữ `connection.go` (minimal changes)
- [ ] Tạo `transform.go` (gộp từ transform domain)
- [ ] Tạo `helpers.go` (validation, parsing, clamping)
- [ ] Tạo `types.go` (private structs)

### Phase 5: Refactor `websocket/delivery/redis/`

- [ ] Tạo `new.go` (Subscriber interface + factory)
- [ ] Refactor `subscriber.go` (thin: Start, listen, reconnect, Shutdown)
- [ ] Tạo `workers.go` (handleMessage → uc.ProcessMessage)
- [ ] Tạo `presenters.go` (RedisMessage DTO)

### Phase 6: Refactor `websocket/delivery/http/`

- [ ] Tạo `new.go` (Handler interface + factory)
- [ ] Refactor `handlers.go` (thin controller)
- [ ] Tạo `process_request.go` (token extraction, filter parsing)
- [ ] Tạo `presenters.go` (WSConfig, CookieConfig, UpgradeReq DTOs)
- [ ] Tạo `routes.go` (RegisterRoutes)
- [ ] Tạo `errors.go` (mapError)

### Phase 7: Delete `internal/transform/`

- [ ] Xóa toàn bộ `internal/transform/` directory

### Phase 8: Update wiring

- [ ] Update `internal/httpserver/handler.go` (new wiring)
- [ ] Update `internal/httpserver/new.go` (remove old Hub/wsHandler fields)
- [ ] Update `cmd/api/main.go` (if needed)

### Phase 9: Tests

- [ ] Update existing tests
- [ ] Add unit tests for new transform logic
- [ ] Add unit tests for alert dispatching
- [ ] Build + vet + test pass

---

## 8. Risk & Mitigation

| Risk                                    | Impact | Mitigation                                                       |
| --------------------------------------- | ------ | ---------------------------------------------------------------- |
| Breaking existing WebSocket connections | HIGH   | Giữ nguyên hub.go core logic, chỉ refactor wrapping              |
| Redis subscriber downtime               | HIGH   | Refactor subscriber delivery thin, keep Start()/listen() pattern |
| Import cycle                            | MEDIUM | Strict dependency: delivery → interface, usecase → interface     |
| Test breakage                           | MEDIUM | Migrate tests phase by phase, keep integration tests last        |

---

**Document Version:** v1.0  
**Author:** AI Assistant  
**Status:** READY FOR REVIEW
