# WebSocket Service Migration Guide: Phase-Based Progress

**Ngày tạo:** 2025-12-15  
**Liên quan:** [integration-project-websocket.md](./integration-project-websocket.md)  
**Độ ưu tiên:** Medium (backward compatible)

---

## 1. Tổng Quan

Project Service đã cập nhật format của messages được publish lên Redis Pub/Sub. WebSocket Service cần cập nhật để:

1. Subscribe đúng channel pattern mới
2. Parse message format mới với phase-based progress
3. Forward message đến clients với structure mới

**Lưu ý quan trọng:** Thay đổi này BACKWARD COMPATIBLE. WebSocket Service có thể hoạt động mà không cần thay đổi code, nhưng nên cập nhật để tận dụng data mới.

---

## 2. Thay Đổi Channel Pattern

### 2.1. Channel Pattern Cũ

```
user_noti:{user_id}
```

**Ví dụ:** `user_noti:user_123`

### 2.2. Channel Pattern Mới

```
project:{project_id}:{user_id}
job:{job_id}:{user_id}
```

**Ví dụ:**

- `project:proj_abc:user_123` - Progress updates cho project
- `job:job_xyz:user_123` - Dry-run job results

### 2.3. Migration Steps

**Bước 1:** Cập nhật subscription pattern

```go
// OLD: Subscribe theo user
func (s *subscriber) SubscribeUser(userID string) {
    channel := fmt.Sprintf("user_noti:%s", userID)
    s.redis.Subscribe(ctx, channel)
}

// NEW: Subscribe theo pattern để nhận tất cả messages của user
func (s *subscriber) SubscribeUser(userID string) {
    // Subscribe pattern để nhận cả project và job messages
    patterns := []string{
        fmt.Sprintf("project:*:%s", userID),  // All project updates
        fmt.Sprintf("job:*:%s", userID),       // All job updates
    }
    s.redis.PSubscribe(ctx, patterns...)
}
```

**Bước 2:** Hoặc subscribe specific project/job

```go
// Subscribe cho specific project
func (s *subscriber) SubscribeProject(projectID, userID string) {
    channel := fmt.Sprintf("project:%s:%s", projectID, userID)
    s.redis.Subscribe(ctx, channel)
}

// Subscribe cho specific job
func (s *subscriber) SubscribeJob(jobID, userID string) {
    channel := fmt.Sprintf("job:%s:%s", jobID, userID)
    s.redis.Subscribe(ctx, channel)
}
```

---

## 3. Thay Đổi Message Format

### 3.1. Project Progress Message

#### Format Cũ (Deprecated)

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_xyz",
    "status": "CRAWLING",
    "progress": {
      "current": 50,
      "total": 100,
      "percentage": 50.0,
      "errors": ["Error 1", "Error 2"]
    }
  }
}
```

#### Format Mới

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_xyz",
    "status": "PROCESSING",
    "crawl": {
      "total": 100,
      "done": 100,
      "errors": 0,
      "progress_percent": 100.0
    },
    "analyze": {
      "total": 100,
      "done": 45,
      "errors": 2,
      "progress_percent": 47.0
    },
    "overall_progress_percent": 73.5
  }
}
```

### 3.2. Project Completed Message

#### Format Mới

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

### 3.3. Message Types

| Type                | Khi nào                     | Status values            |
| ------------------- | --------------------------- | ------------------------ |
| `project_progress`  | Progress update (throttled) | INITIALIZING, PROCESSING |
| `project_completed` | Project finished            | DONE, FAILED             |
| `dryrun_result`     | Dry-run job completed       | success, failed          |

---

## 4. Code Changes Chi Tiết

### 4.1. TypeScript/JavaScript Types

```typescript
// types/websocket.ts

// Phase progress structure
interface PhaseProgress {
  total: number;
  done: number;
  errors: number;
  progress_percent: number;
}

// Project progress payload (NEW)
interface ProjectProgressPayload {
  project_id: string;
  status: "INITIALIZING" | "PROCESSING" | "DONE" | "FAILED";
  crawl: PhaseProgress;
  analyze: PhaseProgress;
  overall_progress_percent: number;
}

// WebSocket message types
type MessageType = "project_progress" | "project_completed" | "dryrun_result";

interface WebSocketMessage<T = unknown> {
  type: MessageType;
  payload: T;
}

// Specific message types
type ProjectProgressMessage = WebSocketMessage<ProjectProgressPayload>;
type ProjectCompletedMessage = WebSocketMessage<ProjectProgressPayload>;
```

### 4.2. Go Types (WebSocket Service)

```go
// internal/types/websocket.go

// PhaseProgress represents progress for a single phase
type PhaseProgress struct {
    Total           int64   `json:"total"`
    Done            int64   `json:"done"`
    Errors          int64   `json:"errors"`
    ProgressPercent float64 `json:"progress_percent"`
}

// ProjectProgressPayload represents the payload for project progress messages
type ProjectProgressPayload struct {
    ProjectID              string        `json:"project_id"`
    Status                 string        `json:"status"`
    Crawl                  PhaseProgress `json:"crawl"`
    Analyze                PhaseProgress `json:"analyze"`
    OverallProgressPercent float64       `json:"overall_progress_percent"`
}

// WebSocketMessage represents a message to be sent to clients
type WebSocketMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}
```

### 4.3. Message Handler Update

```go
// internal/handler/redis_subscriber.go

func (h *Handler) HandleRedisMessage(channel string, message []byte) error {
    // Parse the message
    var wsMessage WebSocketMessage
    if err := json.Unmarshal(message, &wsMessage); err != nil {
        return fmt.Errorf("failed to parse message: %w", err)
    }

    // Extract user ID from channel
    // Channel format: "project:{project_id}:{user_id}" or "job:{job_id}:{user_id}"
    userID := h.extractUserIDFromChannel(channel)
    if userID == "" {
        return fmt.Errorf("invalid channel format: %s", channel)
    }

    // Route message based on type
    switch wsMessage.Type {
    case "project_progress", "project_completed":
        return h.handleProjectMessage(userID, wsMessage)
    case "dryrun_result":
        return h.handleDryRunMessage(userID, wsMessage)
    default:
        h.logger.Warn("unknown message type", "type", wsMessage.Type)
        return nil
    }
}

func (h *Handler) extractUserIDFromChannel(channel string) string {
    // Channel format: "project:{project_id}:{user_id}" or "job:{job_id}:{user_id}"
    parts := strings.Split(channel, ":")
    if len(parts) != 3 {
        return ""
    }
    return parts[2] // user_id is the last part
}

func (h *Handler) handleProjectMessage(userID string, msg WebSocketMessage) error {
    // Get all connections for this user
    connections := h.connectionManager.GetUserConnections(userID)

    // Serialize message once
    data, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to serialize message: %w", err)
    }

    // Send to all user connections
    for _, conn := range connections {
        if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
            h.logger.Error("failed to send message", "user_id", userID, "error", err)
            // Continue sending to other connections
        }
    }

    return nil
}
```

### 4.4. Redis Subscriber Update

```go
// internal/subscriber/redis.go

type RedisSubscriber struct {
    client  *redis.Client
    handler MessageHandler
    logger  Logger
}

func (s *RedisSubscriber) SubscribeUserChannels(ctx context.Context, userID string) error {
    // Use pattern subscription to receive all messages for this user
    patterns := []string{
        fmt.Sprintf("project:*:%s", userID),
        fmt.Sprintf("job:*:%s", userID),
    }

    pubsub := s.client.PSubscribe(ctx, patterns...)
    defer pubsub.Close()

    // Listen for messages
    ch := pubsub.Channel()
    for msg := range ch {
        if err := s.handler.HandleRedisMessage(msg.Channel, []byte(msg.Payload)); err != nil {
            s.logger.Error("failed to handle message",
                "channel", msg.Channel,
                "error", err,
            )
        }
    }

    return nil
}

// Alternative: Subscribe to specific project
func (s *RedisSubscriber) SubscribeProject(ctx context.Context, projectID, userID string) error {
    channel := fmt.Sprintf("project:%s:%s", projectID, userID)

    pubsub := s.client.Subscribe(ctx, channel)
    defer pubsub.Close()

    ch := pubsub.Channel()
    for msg := range ch {
        if err := s.handler.HandleRedisMessage(msg.Channel, []byte(msg.Payload)); err != nil {
            s.logger.Error("failed to handle message",
                "channel", msg.Channel,
                "error", err,
            )
        }
    }

    return nil
}
```

---

## 5. Frontend Client Changes

### 5.1. WebSocket Message Handler

```typescript
// services/websocket.ts

class WebSocketService {
  private socket: WebSocket;
  private handlers: Map<string, (payload: any) => void> = new Map();

  constructor(url: string) {
    this.socket = new WebSocket(url);
    this.socket.onmessage = this.handleMessage.bind(this);
  }

  private handleMessage(event: MessageEvent) {
    try {
      const message: WebSocketMessage = JSON.parse(event.data);

      switch (message.type) {
        case "project_progress":
        case "project_completed":
          this.handleProjectProgress(message.payload as ProjectProgressPayload);
          break;
        case "dryrun_result":
          this.handleDryRunResult(message.payload);
          break;
        default:
          console.warn("Unknown message type:", message.type);
      }
    } catch (error) {
      console.error("Failed to parse WebSocket message:", error);
    }
  }

  private handleProjectProgress(payload: ProjectProgressPayload) {
    // Emit to registered handlers
    const handler = this.handlers.get("project_progress");
    if (handler) {
      handler(payload);
    }

    // Check if completed
    if (payload.status === "DONE" || payload.status === "FAILED") {
      const completedHandler = this.handlers.get("project_completed");
      if (completedHandler) {
        completedHandler(payload);
      }
    }
  }

  // Register handlers
  onProjectProgress(handler: (payload: ProjectProgressPayload) => void) {
    this.handlers.set("project_progress", handler);
  }

  onProjectCompleted(handler: (payload: ProjectProgressPayload) => void) {
    this.handlers.set("project_completed", handler);
  }
}
```

### 5.2. React Component Example

```tsx
// components/ProjectProgress.tsx

import React, { useEffect, useState } from "react";

interface PhaseProgress {
  total: number;
  done: number;
  errors: number;
  progress_percent: number;
}

interface ProjectProgressPayload {
  project_id: string;
  status: string;
  crawl: PhaseProgress;
  analyze: PhaseProgress;
  overall_progress_percent: number;
}

interface Props {
  projectId: string;
}

export function ProjectProgress({ projectId }: Props) {
  const [progress, setProgress] = useState<ProjectProgressPayload | null>(null);

  useEffect(() => {
    // Subscribe to WebSocket updates
    const ws = new WebSocketService(WS_URL);

    ws.onProjectProgress((payload) => {
      if (payload.project_id === projectId) {
        setProgress(payload);
      }
    });

    return () => ws.close();
  }, [projectId]);

  if (!progress) {
    return <div>Loading...</div>;
  }

  return (
    <div className="project-progress">
      <h3>Project Progress</h3>

      {/* Status */}
      <div className="status">
        Status:{" "}
        <span className={`status-${progress.status.toLowerCase()}`}>
          {getStatusLabel(progress.status)}
        </span>
      </div>

      {/* Crawl Phase */}
      <div className="phase">
        <div className="phase-header">
          <span>Thu thập dữ liệu</span>
          <span>
            {progress.crawl.done}/{progress.crawl.total}
          </span>
        </div>
        <div className="progress-bar">
          <div
            className="progress-fill"
            style={{ width: `${progress.crawl.progress_percent}%` }}
          />
        </div>
        {progress.crawl.errors > 0 && (
          <span className="errors">{progress.crawl.errors} lỗi</span>
        )}
      </div>

      {/* Analyze Phase */}
      <div className="phase">
        <div className="phase-header">
          <span>Phân tích</span>
          <span>
            {progress.analyze.done}/{progress.analyze.total}
          </span>
        </div>
        <div className="progress-bar">
          <div
            className="progress-fill"
            style={{ width: `${progress.analyze.progress_percent}%` }}
          />
        </div>
        {progress.analyze.errors > 0 && (
          <span className="errors">{progress.analyze.errors} lỗi</span>
        )}
      </div>

      {/* Overall Progress */}
      <div className="overall">
        <div className="overall-header">
          <span>Tổng tiến độ</span>
          <span>{progress.overall_progress_percent.toFixed(1)}%</span>
        </div>
        <div className="progress-bar overall-bar">
          <div
            className="progress-fill"
            style={{ width: `${progress.overall_progress_percent}%` }}
          />
        </div>
      </div>
    </div>
  );
}

function getStatusLabel(status: string): string {
  switch (status) {
    case "INITIALIZING":
      return "Đang khởi tạo...";
    case "PROCESSING":
      return "Đang xử lý...";
    case "DONE":
      return "Hoàn thành";
    case "FAILED":
      return "Thất bại";
    default:
      return status;
  }
}
```

---

## 6. Migration Checklist

### WebSocket Service (Backend)

- [ ] Update Redis subscription pattern từ `user_noti:{user_id}` sang `project:*:{user_id}` và `job:*:{user_id}`
- [ ] Update channel parsing logic để extract user_id từ new format
- [ ] Add TypeScript/Go types cho PhaseProgress và ProjectProgressPayload
- [ ] Update message handler để parse new payload structure
- [ ] Test với cả old và new message formats
- [ ] Deploy và monitor logs

### Frontend

- [ ] Update TypeScript types cho WebSocket messages
- [ ] Update WebSocket message handler
- [ ] Update progress display components
- [ ] Add phase-based progress UI (crawl + analyze)
- [ ] Test với real WebSocket messages
- [ ] Deploy và verify UI

---

## 7. Testing

### 7.1. Test Message Format

```bash
# Publish test message to Redis
redis-cli PUBLISH "project:test_proj:test_user" '{
  "type": "project_progress",
  "payload": {
    "project_id": "test_proj",
    "status": "PROCESSING",
    "crawl": {"total": 100, "done": 80, "errors": 2, "progress_percent": 82.0},
    "analyze": {"total": 78, "done": 45, "errors": 1, "progress_percent": 59.0},
    "overall_progress_percent": 70.5
  }
}'
```

### 7.2. Test Completion Message

```bash
redis-cli PUBLISH "project:test_proj:test_user" '{
  "type": "project_completed",
  "payload": {
    "project_id": "test_proj",
    "status": "DONE",
    "crawl": {"total": 100, "done": 98, "errors": 2, "progress_percent": 100.0},
    "analyze": {"total": 98, "done": 95, "errors": 3, "progress_percent": 100.0},
    "overall_progress_percent": 100.0
  }
}'
```

### 7.3. Verify Subscription

```bash
# Subscribe to pattern and verify messages are received
redis-cli PSUBSCRIBE "project:*:test_user"
```

---

## 8. Rollback Plan

Nếu có vấn đề sau khi deploy:

1. **WebSocket Service:** Revert code changes, redeploy
2. **Frontend:** Revert to old progress display
3. **Project Service:** Không cần rollback (backward compatible)

**Note:** Project Service vẫn gửi messages với format mới, nhưng WebSocket Service có thể forward raw messages mà không cần parse payload structure.

---

## 9. Timeline Đề Xuất

| Phase | Task                                      | Duration |
| ----- | ----------------------------------------- | -------- |
| 1     | Update WebSocket Service types & handlers | 1 day    |
| 2     | Update Frontend types & components        | 1 day    |
| 3     | Integration testing                       | 0.5 day  |
| 4     | Deploy to staging                         | 0.5 day  |
| 5     | Production deployment                     | 0.5 day  |

**Total:** ~3.5 days

---

## 10. Contact

Nếu có câu hỏi về migration này, liên hệ:

- Project Service team: [contact info]
- Related docs: `document/integration-project-websocket.md`
