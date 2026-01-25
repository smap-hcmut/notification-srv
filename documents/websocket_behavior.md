# WebSocket Service Behavior Specification

**Last Updated**: 2025-12-15  
**Service Version**: 1.1.0  
**Status**: Production Ready

---

## Table of Contents

1. [Overview](#overview)
2. [Core Components](#core-components)
3. [Connection Lifecycle](#connection-lifecycle)
4. [Message Handling](#message-handling)
5. [Phase-Based Progress](#phase-based-progress)
6. [Redis Pub/Sub Subscription](#redis-pubsub-subscription)
7. [Health Monitoring](#health-monitoring)
8. [Error Handling](#error-handling)
9. [Configuration](#configuration)
10. [API Reference](#api-reference)

---

## Overview

### Purpose

WebSocket Service là một real-time notification hub, chịu trách nhiệm:

- Duy trì kết nối WebSocket với clients
- Subscribe Redis Pub/Sub để nhận notifications
- Route messages đến đúng user connections
- Hỗ trợ multiple tabs/devices per user
- Cung cấp health monitoring endpoint

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        WebSocket Service                             │
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │   Handler    │───▶│     Hub      │◀───│     Subscriber       │  │
│  │  (Gin/WS)    │    │ (Connection  │    │  (Redis Pub/Sub)     │  │
│  │              │    │   Manager)   │    │                      │  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
│         │                   │                      │                │
│         │                   │                      │                │
│         ▼                   ▼                      ▼                │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │  Connection  │    │   Message    │    │    RedisMessage      │  │
│  │   (per WS)   │    │   Types      │    │      Parser          │  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
         │                                           │
         ▼                                           ▼
    ┌─────────┐                               ┌─────────────┐
    │ Clients │                               │    Redis    │
    │ (Browser)│                               │  (Pub/Sub)  │
    └─────────┘                               └─────────────┘
```

---

## Core Components

### 1. Handler (`internal/websocket/handler.go`)

**Responsibilities**:

- Xử lý HTTP upgrade requests
- Authenticate users via JWT (cookie hoặc query param)
- Validate CORS origins
- Create và register connections

**Key Behaviors**:

| Behavior            | Description                                                                      |
| ------------------- | -------------------------------------------------------------------------------- |
| JWT Authentication  | Cookie `access_token` (primary) hoặc query param `token` (fallback)              |
| CORS Validation     | Production: static whitelist; Dev/Staging: dynamic (localhost + private subnets) |
| Connection Creation | Mỗi WebSocket connection được wrap trong `Connection` struct                     |

### 2. Hub (`internal/websocket/hub.go`)

**Responsibilities**:

- Quản lý tất cả active connections
- Route messages đến đúng user
- Handle connection registration/unregistration
- Notify subscriber khi user disconnect

**Key Behaviors**:

| Behavior            | Description                                                       |
| ------------------- | ----------------------------------------------------------------- |
| Multi-tab Support   | `map[string][]*Connection` - mỗi user có thể có nhiều connections |
| Max Connections     | Configurable limit, reject new connections khi đạt limit          |
| Graceful Shutdown   | Close tất cả connections khi service shutdown                     |
| Disconnect Callback | Notify `RedisNotifier` khi user disconnect                        |

**Connection Map Structure**:

```go
connections map[string][]*Connection
// Example:
// "user_123" -> [conn1, conn2, conn3]  // 3 tabs open
// "user_456" -> [conn1]                 // 1 tab open
```

### 3. Connection (`internal/websocket/connection.go`)

**Responsibilities**:

- Manage single WebSocket connection
- Handle ping/pong heartbeat
- Read/write message pumps
- Buffer outgoing messages

**Key Behaviors**:

| Behavior     | Description                                                          |
| ------------ | -------------------------------------------------------------------- |
| Ping/Pong    | Server sends ping every `PingPeriod`, expects pong within `PongWait` |
| Write Buffer | Channel-based buffer để tránh blocking                               |
| Read Pump    | Goroutine đọc messages từ client (hiện tại chỉ handle pong)          |
| Write Pump   | Goroutine gửi messages đến client                                    |

### 4. Subscriber (`internal/redis/subscriber.go`)

**Responsibilities**:

- Subscribe Redis Pub/Sub pattern `user_noti:*`
- Parse incoming messages
- Route messages đến Hub
- Track health metrics

**Key Behaviors**:

| Behavior             | Description                                               |
| -------------------- | --------------------------------------------------------- |
| Pattern Subscription | `PSUBSCRIBE user_noti:*` - nhận tất cả user notifications |
| Auto Reconnect       | Retry với exponential backoff khi connection lost         |
| Health Tracking      | Track `lastMessageAt` và `isActive` status                |
| Message Routing      | Extract `userID` từ channel name, gửi đến Hub             |

---

## Connection Lifecycle

### 1. Connection Establishment

```
Client                    Handler                   Hub
  │                          │                       │
  │ GET /ws?token=xxx        │                       │
  │ (or Cookie: access_token)│                       │
  │─────────────────────────▶│                       │
  │                          │                       │
  │                          │ Validate JWT          │
  │                          │ Extract userID        │
  │                          │                       │
  │                          │ Upgrade to WebSocket  │
  │◀─────────────────────────│                       │
  │                          │                       │
  │                          │ Create Connection     │
  │                          │──────────────────────▶│
  │                          │                       │
  │                          │ Notify RedisNotifier  │
  │                          │ OnUserConnected()     │
  │                          │                       │
  │                          │ Start read/write pumps│
  │                          │                       │
  │ Connection established   │                       │
  │◀═════════════════════════│                       │
```

### 2. Message Flow

```
Redis Pub/Sub          Subscriber              Hub              Connection         Client
     │                     │                    │                    │               │
     │ PMESSAGE            │                    │                    │               │
     │ user_noti:user_123  │                    │                    │               │
     │ {"type":"..."}      │                    │                    │               │
     │────────────────────▶│                    │                    │               │
     │                     │                    │                    │               │
     │                     │ Parse message      │                    │               │
     │                     │ Extract userID     │                    │               │
     │                     │                    │                    │               │
     │                     │ SendToUser()       │                    │               │
     │                     │───────────────────▶│                    │               │
     │                     │                    │                    │               │
     │                     │                    │ Find connections   │               │
     │                     │                    │ for user_123       │               │
     │                     │                    │                    │               │
     │                     │                    │ Send to all tabs   │               │
     │                     │                    │───────────────────▶│               │
     │                     │                    │                    │               │
     │                     │                    │                    │ WebSocket     │
     │                     │                    │                    │ frame         │
     │                     │                    │                    │──────────────▶│
```

### 3. Connection Termination

```
Client                Connection              Hub                Subscriber
  │                       │                    │                     │
  │ Close connection      │                    │                     │
  │──────────────────────▶│                    │                     │
  │                       │                    │                     │
  │                       │ Unregister         │                     │
  │                       │───────────────────▶│                     │
  │                       │                    │                     │
  │                       │                    │ Remove from map     │
  │                       │                    │                     │
  │                       │                    │ Check other conns   │
  │                       │                    │ hasOtherConnections │
  │                       │                    │                     │
  │                       │                    │ OnUserDisconnected()│
  │                       │                    │────────────────────▶│
  │                       │                    │                     │
  │                       │                    │                     │ Update tracking
  │                       │                    │                     │
```

---

## Message Handling

### Message Types

```go
const (
    MessageTypeNotification     MessageType = "notification"
    MessageTypeAlert            MessageType = "alert"
    MessageTypeUpdate           MessageType = "update"
    MessageTypePing             MessageType = "ping"
    MessageTypePong             MessageType = "pong"
    MessageTypeProjectProgress  MessageType = "project_progress"
    MessageTypeProjectCompleted MessageType = "project_completed"
)
```

### Message Structure

**Generic Message**:

```json
{
  "type": "notification",
  "payload": { ... },
  "timestamp": "2025-12-07T10:00:00Z"
}
```

**Project Progress Message**:

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_xyz",
    "status": "CRAWLING",
    "total": 1000,
    "done": 150,
    "errors": 2,
    "progress_percent": 15.0
  },
  "timestamp": "2025-12-07T10:00:00Z"
}
```

**Project Completed Message**:

```json
{
  "type": "project_completed",
  "payload": {
    "project_id": "proj_xyz",
    "status": "DONE",
    "total": 1000,
    "done": 1000,
    "errors": 5,
    "progress_percent": 100.0
  },
  "timestamp": "2025-12-07T10:00:00Z"
}
```

### Payload Validation

```go
type ProgressPayload struct {
    ProjectID       string  `json:"project_id"`       // Required, non-empty
    Status          string  `json:"status"`           // CRAWLING, DONE, FAILED
    Total           int     `json:"total"`            // >= 0
    Done            int     `json:"done"`             // >= 0
    Errors          int     `json:"errors"`           // >= 0
    ProgressPercent float64 `json:"progress_percent"` // 0.0 - 100.0
}
```

---

## Phase-Based Progress

> **New in v1.1.0**: Hỗ trợ phase-based progress với chi tiết tiến độ cho từng phase (crawl, analyze).

### Overview

Phase-Based Progress cho phép frontend hiển thị tiến độ chi tiết cho từng phase của project:

```
┌─────────────────────────────────────────────────────┐
│ Project: proj_xyz                                   │
│ Status: PROCESSING                                  │
│ ┌─────────────────────────────────────────────────┐ │
│ │ Crawl Phase     [████████░░] 82% (80/100)       │ │
│ │ Analyze Phase   [█████░░░░░] 59% (45/78)        │ │
│ └─────────────────────────────────────────────────┘ │
│ Overall: 70.5%                                      │
└─────────────────────────────────────────────────────┘
```

### Message Format Detection

Transform layer tự động detect format dựa vào `type` field:

| Condition | Format |
|-----------|--------|
| `type` = `project_progress` hoặc `project_completed` | Phase-Based |
| Không có `type` field hoặc giá trị khác | Legacy |

### Phase-Based Message Structure

**Progress Update:**

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_xyz",
    "status": "PROCESSING",
    "crawl": {
      "total": 100,
      "done": 80,
      "errors": 2,
      "progress_percent": 82.0
    },
    "analyze": {
      "total": 78,
      "done": 45,
      "errors": 1,
      "progress_percent": 59.0
    },
    "overall_progress_percent": 70.5
  }
}
```

**Completion Notification:**

```json
{
  "type": "project_completed",
  "payload": {
    "project_id": "proj_xyz",
    "status": "DONE",
    "crawl": {
      "total": 100,
      "done": 98,
      "errors": 2,
      "progress_percent": 100.0
    },
    "analyze": {
      "total": 98,
      "done": 95,
      "errors": 3,
      "progress_percent": 100.0
    },
    "overall_progress_percent": 100.0
  }
}
```

### Phase-Based Status Values

| Status | Description |
|--------|-------------|
| `INITIALIZING` | Project đang được khởi tạo |
| `PROCESSING` | Crawl và/hoặc analyze phase đang chạy |
| `DONE` | Tất cả phases hoàn thành thành công |
| `FAILED` | Processing failed |

### Phase Progress Fields

| Field | Type | Description |
|-------|------|-------------|
| `total` | int64 | Tổng số items trong phase |
| `done` | int64 | Số items đã hoàn thành |
| `errors` | int64 | Số items bị lỗi |
| `progress_percent` | float64 | Phần trăm tiến độ (0.0-100.0) |

### Transform Layer Processing

```go
// Auto-detect và route đúng transformer
func (t *ProjectTransformer) TransformAny(ctx context.Context, payload string, projectID, userID string) (interface{}, error) {
    if types.IsPhaseBasedMessage([]byte(payload)) {
        return t.TransformPhaseBased(ctx, payload, projectID, userID)
    }
    return t.Transform(ctx, payload, projectID, userID)
}
```

### Backward Compatibility

- Legacy format vẫn được hỗ trợ đầy đủ
- Clients cũ có thể ignore các fields mới (`crawl`, `analyze`, `overall_progress_percent`)
- Không cần thay đổi client code để tiếp tục nhận legacy messages

---

## Redis Pub/Sub Subscription

### Channel Pattern

```
user_noti:{user_id}
```

**Examples**:

- `user_noti:user_123` - Notifications for user_123
- `user_noti:user_456` - Notifications for user_456

### Subscription Flow

```go
// Pattern subscription
s.pubsub = s.client.PSubscribe(s.ctx, "user_noti:*")

// Message handling
func (s *Subscriber) handleMessage(channel string, payload string) {
    // 1. Track health
    s.lastMessageAt = time.Now()

    // 2. Extract userID from channel
    // channel = "user_noti:user_123" -> userID = "user_123"
    parts := strings.Split(channel, ":")
    userID := parts[1]

    // 3. Parse payload
    var redisMsg RedisMessage
    json.Unmarshal([]byte(payload), &redisMsg)

    // 4. Create WebSocket message
    wsMsg := &ws.Message{
        Type:      ws.MessageType(redisMsg.Type),
        Payload:   redisMsg.Payload,
        Timestamp: time.Now(),
    }

    // 5. Route to Hub
    s.hub.SendToUser(userID, wsMsg)
}
```

### Reconnection Strategy

```go
maxRetries := 10
retryDelay := 5 * time.Second

for i := 0; i < maxRetries; i++ {
    // Close old connection
    s.pubsub.Close()

    // Create new subscription
    s.pubsub = s.client.PSubscribe(s.ctx, s.patternChannel)

    // Test connection
    if _, err := s.pubsub.Receive(s.ctx); err == nil {
        return nil // Success
    }

    time.Sleep(retryDelay)
}
```

---

## Health Monitoring

### Health Endpoint

**URL**: `GET /health`

**Response**:

```json
{
  "status": "healthy",
  "timestamp": "2025-12-07T10:00:00Z",
  "redis": {
    "status": "connected",
    "ping_ms": 0.5
  },
  "websocket": {
    "active_connections": 150,
    "total_unique_users": 75
  },
  "subscriber": {
    "active": true,
    "last_message_at": "2025-12-07T09:59:55Z",
    "pattern": "user_noti:*"
  },
  "uptime_seconds": 86400
}
```

### Health Status Values

| Status     | HTTP Code | Description                                      |
| ---------- | --------- | ------------------------------------------------ |
| `healthy`  | 200       | All components operational                       |
| `degraded` | 503       | Redis disconnected, service partially functional |

### Subscriber Health Tracking

```go
type Subscriber struct {
    lastMessageAt time.Time   // Updated on each message
    isActive      atomic.Bool // true when running, false on shutdown
}

func (s *Subscriber) GetHealthInfo() (active bool, lastMessageAt time.Time, pattern string) {
    return s.isActive.Load(), s.lastMessageAt, s.patternChannel
}
```

---

## Error Handling

### Connection Errors

| Error                    | Handling                       |
| ------------------------ | ------------------------------ |
| JWT Invalid/Expired      | Return 401 Unauthorized        |
| Max Connections Reached  | Reject connection, log warning |
| WebSocket Upgrade Failed | Log error, return HTTP error   |
| Write Buffer Full        | Skip message, log warning      |

### Redis Errors

| Error                  | Handling                  |
| ---------------------- | ------------------------- |
| Connection Lost        | Auto reconnect with retry |
| Parse Error            | Log error, skip message   |
| Channel Format Invalid | Log warning, skip message |

### Graceful Shutdown

```go
func (h *Hub) Shutdown(ctx context.Context) error {
    h.cancel() // Signal shutdown

    select {
    case <-h.done:
        return nil // Clean shutdown
    case <-ctx.Done():
        return ctx.Err() // Timeout
    }
}

func (s *Subscriber) Shutdown(ctx context.Context) error {
    s.isActive.Store(false)
    s.cancel()
    s.pubsub.Close()

    select {
    case <-s.done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

---

## Configuration

### Environment Variables

| Variable                    | Default        | Description                |
| --------------------------- | -------------- | -------------------------- |
| `SERVER_HOST`               | `0.0.0.0`      | Server bind address        |
| `SERVER_PORT`               | `8081`         | Server port                |
| `REDIS_HOST`                | `localhost`    | Redis host                 |
| `REDIS_PORT`                | `6379`         | Redis port                 |
| `WEBSOCKET_MAX_CONNECTIONS` | `10000`        | Max concurrent connections |
| `WEBSOCKET_PONG_WAIT`       | `60s`          | Pong timeout               |
| `WEBSOCKET_PING_INTERVAL`   | `54s`          | Ping interval              |
| `WEBSOCKET_WRITE_WAIT`      | `10s`          | Write timeout              |
| `JWT_SECRET_KEY`            | -              | JWT signing key            |
| `COOKIE_NAME`               | `access_token` | Auth cookie name           |
| `ENVIRONMENT_NAME`          | `production`   | Environment for CORS       |

### CORS Configuration

**Production Mode**:

```go
allowedOrigins := []string{
    "https://smap.tantai.dev",
    "https://smap-api.tantai.dev",
}
```

**Dev/Staging Mode**:

```go
// Additional allowed:
// - localhost:*
// - 127.0.0.1:*
// - Private subnets: 172.16.21.0/24, 172.16.19.0/24, 192.168.1.0/24
```

---

## API Reference

### WebSocket Endpoint

**URL**: `GET /ws`

**Authentication**:

- Cookie: `access_token={jwt_token}` (preferred)
- Query: `?token={jwt_token}` (fallback, deprecated)

**Response**:

- Success: WebSocket upgrade (101 Switching Protocols)
- Error: JSON error response

**Error Responses**:

```json
// 401 Unauthorized - Missing token
{"error": "missing token parameter"}

// 401 Unauthorized - Invalid token
{"error": "invalid or expired token"}
```

### Health Endpoint

**URL**: `GET /health`

**Response**: See [Health Monitoring](#health-monitoring)

### Metrics Endpoint

**URL**: `GET /metrics`

**Response**:

```json
{
  "active_connections": 150,
  "total_unique_users": 75,
  "total_messages_sent": 10000,
  "total_messages_received": 5000,
  "total_messages_failed": 10
}
```
