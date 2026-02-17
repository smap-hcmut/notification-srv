# WebSocket Domain Code Plan

**Ref:** `documents/master-proposal.md`, `documents/domain_convention/convention.md`
**Status:** DRAFT
**Domain:** `internal/websocket`

This document details the implementation plan for the `websocket` domain based on the master proposal and adhering to the project's domain conventions.

---

## 1. Directory Structure

```
internal/websocket/
├── delivery/
│   ├── http/
│   │   ├── new.go                 # Factory: NewHandler
│   │   ├── handlers.go            # Controller: HandleWebSocket
│   │   ├── process_request.go     # Input processing: UpgradeReq -> ConnectionInput
│   │   ├── presenters.go          # DTOs: UpgradeReq, Channel Configs
│   │   ├── routes.go              # Route definitions
│   │   └── errors.go              # Error mapping: Domain -> HTTP
│   └── redis/
│       ├── new.go                 # Factory: NewSubscriber
│       ├── subscriber.go          # Manager: Start, Subscribe, Reconnect
│       ├── workers.go             # Logic: handleMessage (Parse -> UseCase)
│       └── presenters.go          # DTOs: RedisMessage, Channel Parsing
├── usecase/
│   ├── new.go                     # Factory: NewUseCase
│   ├── hub.go                     # Logic: Hub management (Run, Register, Unregister)
│   ├── connection.go              # Logic: Connection pump (Read/Write)
│   ├── transform.go               # Logic: Message processing pipeline
│   ├── helpers.go                 # Private helpers: Validation, Formatting
│   └── types.go                   # Private structs (internal to usecase)
├── interface.go                   # Public UseCase interface
├── types.go                       # Public UseCase Input/Output structs
└── errors.go                      # Domain sentinel errors
```

---

## 2. Public Interfaces & Types (`root`)

### 2.1 `internal/websocket/interface.go`

```go
package websocket

import (
 "context"
)

// UseCase defines the business logic for the WebSocket domain.
// It combines connection management and message processing/transformation.
type UseCase interface {
 // Lifecycle
 Run()
 Shutdown(ctx context.Context) error

 // Connection Management
 // Note: Register takes a Connection interface/struct defined in types.go or internal
 Register(ctx context.Context, input ConnectionInput) error
 Unregister(ctx context.Context, input ConnectionInput) error

 // Stats
 GetStats(ctx context.Context) (HubStats, error)

 // Message Processing (Call by Redis Delivery or HTTP)
 // Validates, Transforms, and Routes message to connected users
 ProcessMessage(ctx context.Context, input ProcessMessageInput) error

 // Event Callbacks (Call by Redis Delivery)
 OnUserConnected(ctx context.Context, userID string) error
 OnUserDisconnected(ctx context.Context, userID string, hasOtherConnections bool) error
}
```

### 2.2 `internal/websocket/types.go`

```go
package websocket

import "time"

// --- Message Types ---
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

// --- UseCase Inputs ---

// ProcessMessageInput is the raw input from Redis
type ProcessMessageInput struct {
 Channel string
 Payload []byte
}

// ConnectionInput represents a new connection attempt
type ConnectionInput struct {
 UserID    string
 ProjectID string      // Optional filter
 Conn      interface{} // *websocket.Conn (handled as interface{} to avoid direct dependency in public type if preferred, or wrapped)
}

// --- UseCase Outputs ---

type HubStats struct {
 ActiveConnections int
 TotalUniqueUsers  int
}

// NotificationOutput is the final payload sent to the client
type NotificationOutput struct {
 Type      MessageType `json:"type"`
 Timestamp time.Time   `json:"timestamp"`
 Payload   interface{} `json:"payload"`
}
```

### 2.3 `internal/websocket/errors.go`

```go
package websocket

import "errors"

var (
 ErrInvalidToken          = errors.New("invalid or expired JWT token")
 ErrConnectionClosed      = errors.New("connection closed")
 ErrMaxConnectionsReached = errors.New("maximum connections reached")
 ErrInvalidMessage        = errors.New("invalid message format")
 ErrTransformFailed       = errors.New("message transformation failed")
)
```

---

## 3. Delivery Layer Implementation

### 3.1 HTTP Delivery (`delivery/http`)

**`handlers.go`**

- Coordinates the WebSocket upgrade.
- calls `processUpgradeRequest` to validate token/params.
- calls `uc.Register`.
- Note: The actual `conn` object is created here using `gorilla/websocket`.

**`process_request.go`**

- logic to extract JWT from query/cookie.
- logic to validate `project_id` filter.
- Returns `UpgradeReq` DTO.

**`presenters.go`**

```go
type UpgradeReq struct {
    Token     string `form:"token"`
    ProjectID string `form:"project_id"`
}

func (r UpgradeReq) validate() error { ... }
func (r UpgradeReq) toInput(conn *websocket.Conn, userID string) websocket.ConnectionInput { ... }
```

**`routes.go`**

```go
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
    ws := r.Group("/ws")
    {
        ws.GET("", h.HandleWebSocket)
    }
}
```

### 3.2 Redis Delivery (`delivery/redis`)

**`subscriber.go`**

- Implements `Start()`, `Shutdown()`.
- Connects to Redis Pub/Sub.
- Listens in a loop.

**`workers.go`**

- `handleMessage(ctx, msg *redis.Message)`
- calls `presenters.parseMessage(msg)`
- calls `uc.ProcessMessage(ctx, input)`

**`presenters.go`**

- Helper functions to parse Redis channel strings (e.g., `project:vac:user:u1`).
- `RedisMessage` DTO (if needed separate from `redis.Message`).

---

## 4. UseCase Layer Implementation

### 4.1 `usecase/new.go`

```go
type implUseCase struct {
 hub            *Hub            // Private, managed internally
 alertUC        alert.UseCase   // Dependency
 logger         log.Logger
 maxConnections int
}

func New(logger log.Logger, maxConnections int, alertUC alert.UseCase) websocket.UseCase {
 return &implUseCase{
  hub:            newHub(logger, maxConnections), // newHub is private
  alertUC:        alertUC,
  logger:         logger,
  maxConnections: maxConnections,
 }
}
```

### 4.2 `usecase/hub.go`

- **Responsibilities**:
  - Manage active connections (`map[*Connection]bool`).
  - Manage user mapping (`map[string]map[*Connection]bool`).
  - Broadcast methods (fan-out).
  - Thread-safe operations (sync.RWMutex).

### 4.3 `usecase/connection.go`

- **Responsibilities**:
  - Defines `Connection` struct (private).
  - `readPump`: Handles PING/PONG and incoming messages (if any expected from client).
  - `writePump`: Flushes `send` channel to WebSocket.

### 4.4 `usecase/transform.go`

- **Responsibilities**:
  - `transformMessage(ctx, msgType, payload)`
  - Main logic to determine specific transformations based on `parsedChannel` and `detectMessageType`.
  - Returns `NotificationOutput`.

### 4.5 `usecase/helpers.go`

- **Responsibilities**:
  - `parseChannel(channel string) (ParsedChannel, error)`
  - `detectMessageType(payload []byte) (MessageType, error)`
  - Validators for payload data.

---

## 5. Dependency Wiring (Draft)

In `cmd/api/main.go` or `internal/httpserver/handler.go`:

```go
// 1. Alert (Domain #2)
alertUC := alert.New(logger, discordClient)

// 2. WebSocket (Domain #1)
wsUC := websocket.New(logger, 10000, alertUC)

// 3. Delivery
redisSub := wsRedis.New(redisClient, wsUC, logger)
wsHandler := wsHttp.New(wsUC, logger)

// 4. Start
go wsUC.Run()
go redisSub.Start()
```

## 6. Implementation Checklist

- [ ] **Infrastructure**: `internal/websocket/interface.go`, `types.go`, `errors.go`
- [ ] **UseCase**: `hub.go`, `connection.go`, `transform.go`
- [ ] **Delivery (HTTP)**: `handlers.go`, `process_request.go`
- [ ] **Delivery (Redis)**: `subscriber.go`, `workers.go`
- [ ] **Wiring**: Update `internal/httpserver`
