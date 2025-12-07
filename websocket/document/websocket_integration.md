# WebSocket Service Integration Guide

**Last Updated**: 2025-12-07  
**Service Version**: 1.0.0  
**Status**: Production Ready

---

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Integration with Project Service](#integration-with-project-service)
4. [Integration with Collector Service](#integration-with-collector-service)
5. [Integration with Frontend](#integration-with-frontend)
6. [Redis Pub/Sub Protocol](#redis-pubsub-protocol)
7. [Message Flow Examples](#message-flow-examples)
8. [Deployment Topology](#deployment-topology)
9. [Troubleshooting](#troubleshooting)

---

## Overview

### Service Role in SMAP System

WebSocket Service đóng vai trò là **Real-time Notification Hub**, kết nối các backend services với frontend clients:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              SMAP System                                     │
│                                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │   Project   │    │  Collector  │    │   Crawler   │    │  Analytics  │  │
│  │   Service   │    │   Service   │    │  Services   │    │   Service   │  │
│  └──────┬──────┘    └──────┬──────┘    └─────────────┘    └─────────────┘  │
│         │                  │                                                 │
│         │ Progress         │ State                                           │
│         │ Callback         │ Updates                                         │
│         ▼                  ▼                                                 │
│  ┌─────────────────────────────────────┐                                    │
│  │              Redis                   │                                    │
│  │  ┌─────────────┐  ┌──────────────┐  │                                    │
│  │  │   DB 0      │  │    DB 1      │  │                                    │
│  │  │  Pub/Sub    │  │ State Store  │  │                                    │
│  │  └──────┬──────┘  └──────────────┘  │                                    │
│  └─────────┼────────────────────────────┘                                    │
│            │                                                                 │
│            │ Subscribe                                                       │
│            ▼                                                                 │
│  ┌─────────────────────────────────────┐                                    │
│  │        WebSocket Service            │                                    │
│  │  ┌───────────┐  ┌────────────────┐  │                                    │
│  │  │Subscriber │──│      Hub       │  │                                    │
│  │  └───────────┘  └───────┬────────┘  │                                    │
│  └─────────────────────────┼───────────┘                                    │
│                            │                                                 │
│                            │ WebSocket                                       │
│                            ▼                                                 │
│  ┌─────────────────────────────────────┐                                    │
│  │           Frontend Clients          │                                    │
│  │  ┌─────────┐ ┌─────────┐ ┌───────┐  │                                    │
│  │  │ Browser │ │ Browser │ │Mobile │  │                                    │
│  │  │  Tab 1  │ │  Tab 2  │ │  App  │  │                                    │
│  │  └─────────┘ └─────────┘ └───────┘  │                                    │
│  └─────────────────────────────────────┘                                    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Integration Points

| Service           | Integration Type     | Direction            | Protocol         |
| ----------------- | -------------------- | -------------------- | ---------------- |
| Project Service   | Redis Pub/Sub        | Project → WebSocket  | Redis PUBLISH    |
| Collector Service | Redis State          | Collector → Redis    | Redis HSET       |
| Frontend          | WebSocket            | WebSocket ↔ Frontend | WS Protocol      |
| Redis             | Pub/Sub Subscription | Redis → WebSocket    | Redis PSUBSCRIBE |

---

## System Architecture

### Data Flow Overview

```
1. User executes project via Project Service
2. Project Service initializes Redis state
3. Project Service publishes project.created event to RabbitMQ
4. Collector Service consumes event, dispatches crawlers
5. Collector Service updates Redis state (total, done, errors)
6. Collector Service calls Project Service progress webhook
7. Project Service publishes to Redis Pub/Sub (user_noti:{user_id})
8. WebSocket Service receives message via subscription
9. WebSocket Service routes message to user's connections
10. Frontend receives real-time progress update
```

### Sequence Diagram

```
Frontend        WebSocket       Redis          Project        Collector       Crawler
   │               │              │               │               │              │
   │ Connect WS    │              │               │               │              │
   │──────────────▶│              │               │               │              │
   │               │              │               │               │              │
   │               │ PSUBSCRIBE   │               │               │              │
   │               │ user_noti:*  │               │               │              │
   │               │─────────────▶│               │               │              │
   │               │              │               │               │              │
   │               │              │               │ Execute       │              │
   │               │              │               │──────────────▶│              │
   │               │              │               │               │              │
   │               │              │               │               │ Dispatch     │
   │               │              │               │               │─────────────▶│
   │               │              │               │               │              │
   │               │              │ HSET state    │               │              │
   │               │              │◀──────────────────────────────│              │
   │               │              │               │               │              │
   │               │              │               │ POST /progress│              │
   │               │              │               │◀──────────────│              │
   │               │              │               │               │              │
   │               │              │ PUBLISH       │               │              │
   │               │              │ user_noti:123 │               │              │
   │               │              │◀──────────────│               │              │
   │               │              │               │               │              │
   │               │ PMESSAGE     │               │               │              │
   │               │◀─────────────│               │               │              │
   │               │              │               │               │              │
   │ WS Message    │              │               │               │              │
   │◀──────────────│              │               │               │              │
   │               │              │               │               │              │
```

---

## Integration with Project Service

### Overview

Project Service là **publisher** chính của notifications. Khi có progress update, Project Service publish message đến Redis Pub/Sub.

### Communication Flow

```
Collector Service                Project Service                    Redis
       │                               │                              │
       │ POST /internal/progress/callback                             │
       │ {project_id, user_id, status, total, done, errors}           │
       │──────────────────────────────▶│                              │
       │                               │                              │
       │                               │ Calculate progress_percent   │
       │                               │                              │
       │                               │ Determine message type       │
       │                               │ (progress vs completed)      │
       │                               │                              │
       │                               │ PUBLISH user_noti:{user_id}  │
       │                               │ {type, payload}              │
       │                               │─────────────────────────────▶│
       │                               │                              │
       │              200 OK           │                              │
       │◀──────────────────────────────│                              │
```

### Project Service Code (Publisher)

```go
// internal/usecase/progress.go

func (uc *usecase) HandleProgressCallback(ctx context.Context, req ProgressCallbackRequest) error {
    channel := fmt.Sprintf("user_noti:%s", req.UserID)

    // Calculate progress percentage
    var progressPercent float64
    if req.Total > 0 {
        progressPercent = float64(req.Done) / float64(req.Total) * 100
    }

    // Determine message type
    messageType := "project_progress"
    if req.Status == "DONE" || req.Status == "FAILED" {
        messageType = "project_completed"
    }

    // Build message
    message := map[string]interface{}{
        "type": messageType,
        "payload": map[string]interface{}{
            "project_id":       req.ProjectID,
            "status":           req.Status,
            "total":            req.Total,
            "done":             req.Done,
            "errors":           req.Errors,
            "progress_percent": progressPercent,
        },
    }

    // Publish to Redis
    data, _ := json.Marshal(message)
    return uc.redisClient.Publish(ctx, channel, string(data)).Err()
}
```

### Message Format from Project Service

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
  }
}
```

### Configuration Requirements

**Project Service** cần:

```env
# Redis (same instance as WebSocket Service)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0  # DB 0 for Pub/Sub
```

---

## Integration with Collector Service

### Overview

Collector Service không trực tiếp communicate với WebSocket Service. Thay vào đó, nó:

1. Updates Redis state (DB 1)
2. Calls Project Service webhook
3. Project Service publishes to Redis Pub/Sub

### Indirect Communication Flow

```
Collector Service          Redis DB 1           Project Service          Redis DB 0
       │                       │                       │                      │
       │ HSET smap:proj:xyz    │                       │                      │
       │ done +1               │                       │                      │
       │──────────────────────▶│                       │                      │
       │                       │                       │                      │
       │ POST /internal/progress/callback              │                      │
       │──────────────────────────────────────────────▶│                      │
       │                       │                       │                      │
       │                       │                       │ PUBLISH user_noti:*  │
       │                       │                       │─────────────────────▶│
       │                       │                       │                      │
       │                       │                       │                      │
       │                       │                       │                      ▼
       │                       │                       │              WebSocket Service
```

### Collector Service Code

```python
# collector/services/progress.py

class ProgressService:
    def __init__(self, redis_client, project_service_client):
        self.redis = redis_client
        self.project_client = project_service_client
        self.throttle_interval = 5  # seconds
        self.last_notify = {}

    def on_item_completed(self, project_id: str, user_id: str, is_error: bool = False):
        # 1. Always update Redis state
        key = f"smap:proj:{project_id}"
        if is_error:
            self.redis.hincrby(key, "errors", 1)
        self.redis.hincrby(key, "done", 1)

        # 2. Throttle webhook calls
        if self._should_notify(project_id):
            state = self._get_state(project_id)
            self.project_client.notify_progress(
                project_id=project_id,
                user_id=user_id,
                status=state["status"],
                total=state["total"],
                done=state["done"],
                errors=state["errors"]
            )

    def _should_notify(self, project_id: str) -> bool:
        now = time.time()
        last = self.last_notify.get(project_id, 0)
        if now - last > self.throttle_interval:
            self.last_notify[project_id] = now
            return True
        return False
```

### Redis State Schema

```
Key: smap:proj:{project_id}
Type: Hash

Fields:
  status: "INITIALIZING" | "CRAWLING" | "PROCESSING" | "DONE" | "FAILED"
  total: int (total items to process)
  done: int (items completed)
  errors: int (items failed)
```

---

## Integration with Frontend

### WebSocket Connection

**URL**: `wss://websocket.tantai.dev/ws`

**Authentication**:

- Primary: Cookie `access_token` (set by API Gateway after login)
- Fallback: Query param `?token={jwt}` (deprecated)

### Frontend Code Example (JavaScript)

```javascript
// services/websocket.js

class WebSocketService {
  constructor() {
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 3000;
    this.listeners = new Map();
  }

  connect() {
    // Cookie-based auth (preferred)
    this.ws = new WebSocket("wss://websocket.tantai.dev/ws");

    // Or with token (fallback)
    // this.ws = new WebSocket(`wss://websocket.tantai.dev/ws?token=${token}`);

    this.ws.onopen = () => {
      console.log("WebSocket connected");
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onclose = () => {
      console.log("WebSocket disconnected");
      this.attemptReconnect();
    };

    this.ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };
  }

  handleMessage(message) {
    const { type, payload, timestamp } = message;

    switch (type) {
      case "project_progress":
        this.emit("progress", payload);
        break;
      case "project_completed":
        this.emit("completed", payload);
        break;
      case "notification":
        this.emit("notification", payload);
        break;
      default:
        console.warn("Unknown message type:", type);
    }
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event).push(callback);
  }

  emit(event, data) {
    const callbacks = this.listeners.get(event) || [];
    callbacks.forEach((cb) => cb(data));
  }

  attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      console.log(
        `Reconnecting... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`
      );
      setTimeout(() => this.connect(), this.reconnectDelay);
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

export default new WebSocketService();
```

### React Hook Example

```javascript
// hooks/useProjectProgress.js

import { useState, useEffect } from "react";
import websocketService from "../services/websocket";

export function useProjectProgress(projectId) {
  const [progress, setProgress] = useState({
    status: "INITIALIZING",
    total: 0,
    done: 0,
    errors: 0,
    progressPercent: 0,
  });

  useEffect(() => {
    const handleProgress = (payload) => {
      if (payload.project_id === projectId) {
        setProgress({
          status: payload.status,
          total: payload.total,
          done: payload.done,
          errors: payload.errors,
          progressPercent: payload.progress_percent,
        });
      }
    };

    const handleCompleted = (payload) => {
      if (payload.project_id === projectId) {
        setProgress({
          status: payload.status,
          total: payload.total,
          done: payload.done,
          errors: payload.errors,
          progressPercent: 100,
        });
      }
    };

    websocketService.on("progress", handleProgress);
    websocketService.on("completed", handleCompleted);

    return () => {
      // Cleanup listeners
    };
  }, [projectId]);

  return progress;
}
```

### Vue.js Example

```javascript
// composables/useWebSocket.js

import { ref, onMounted, onUnmounted } from "vue";

export function useWebSocket() {
  const ws = ref(null);
  const isConnected = ref(false);
  const progress = ref({});

  const connect = () => {
    ws.value = new WebSocket("wss://websocket.tantai.dev/ws");

    ws.value.onopen = () => {
      isConnected.value = true;
    };

    ws.value.onmessage = (event) => {
      const message = JSON.parse(event.data);

      if (
        message.type === "project_progress" ||
        message.type === "project_completed"
      ) {
        progress.value[message.payload.project_id] = message.payload;
      }
    };

    ws.value.onclose = () => {
      isConnected.value = false;
      // Auto reconnect
      setTimeout(connect, 3000);
    };
  };

  onMounted(() => {
    connect();
  });

  onUnmounted(() => {
    if (ws.value) {
      ws.value.close();
    }
  });

  return {
    isConnected,
    progress,
  };
}
```

---

## Redis Pub/Sub Protocol

### Channel Naming Convention

```
user_noti:{user_id}
```

**Examples**:

- `user_noti:user_abc123` - User abc123's notifications
- `user_noti:user_xyz789` - User xyz789's notifications

### Message Schema

```json
{
  "type": "string", // Message type identifier
  "payload": {
    // Type-specific payload
    // ... fields depend on type
  }
}
```

### Supported Message Types

| Type                | Description                     | Payload Fields                                            |
| ------------------- | ------------------------------- | --------------------------------------------------------- |
| `project_progress`  | Progress update during crawling | project_id, status, total, done, errors, progress_percent |
| `project_completed` | Project finished (success/fail) | project_id, status, total, done, errors, progress_percent |
| `notification`      | Generic notification            | title, message, level                                     |
| `alert`             | Alert/warning                   | title, message, severity                                  |

### Publishing Messages (for other services)

```go
// Go example
func PublishNotification(ctx context.Context, redis *redis.Client, userID string, msg interface{}) error {
    channel := fmt.Sprintf("user_noti:%s", userID)
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    return redis.Publish(ctx, channel, string(data)).Err()
}

// Usage
msg := map[string]interface{}{
    "type": "notification",
    "payload": map[string]interface{}{
        "title":   "New Comment",
        "message": "Someone commented on your project",
        "level":   "info",
    },
}
PublishNotification(ctx, redisClient, "user_123", msg)
```

```python
# Python example
def publish_notification(redis_client, user_id: str, msg: dict):
    channel = f"user_noti:{user_id}"
    redis_client.publish(channel, json.dumps(msg))

# Usage
msg = {
    "type": "notification",
    "payload": {
        "title": "New Comment",
        "message": "Someone commented on your project",
        "level": "info"
    }
}
publish_notification(redis_client, "user_123", msg)
```

---

## Message Flow Examples

### Example 1: Project Execution Progress

```
Timeline:
T+0s    User clicks "Execute" on project proj_xyz
T+0.1s  Project Service initializes Redis state
T+0.2s  Project Service publishes project.created to RabbitMQ
T+1s    Collector Service receives event, dispatches crawlers
T+2s    Collector sets total=1000 in Redis
T+2.1s  Collector calls progress webhook (status=CRAWLING, total=1000, done=0)
T+2.2s  Project Service publishes to user_noti:user_123
T+2.3s  WebSocket Service receives message
T+2.4s  Frontend receives: {type: "project_progress", payload: {done: 0, total: 1000}}

T+10s   Crawler completes 100 items
T+10.1s Collector updates Redis (done=100)
T+10.2s Collector calls progress webhook (throttled - 5s passed)
T+10.3s Frontend receives: {type: "project_progress", payload: {done: 100, total: 1000}}

... (continues until done=1000)

T+300s  All items completed
T+300.1s Collector sets status=DONE
T+300.2s Collector calls progress webhook
T+300.3s Frontend receives: {type: "project_completed", payload: {status: "DONE", done: 1000}}
```

### Example 2: Multi-Tab Scenario

```
User has 3 browser tabs open:
- Tab 1: Dashboard
- Tab 2: Project Detail
- Tab 3: Analytics

WebSocket Service Hub state:
connections["user_123"] = [conn_tab1, conn_tab2, conn_tab3]

When progress update arrives:
1. Subscriber receives PMESSAGE user_noti:user_123
2. Hub.SendToUser("user_123", message)
3. Hub broadcasts to ALL 3 connections
4. All 3 tabs receive the same update simultaneously
```

### Example 3: User Disconnect

```
User closes Tab 2:

1. conn_tab2 WebSocket closes
2. Hub.unregisterConnection(conn_tab2)
3. Hub removes conn_tab2 from connections["user_123"]
4. hasOtherConnections = true (Tab 1 and 3 still open)
5. Hub calls redisNotifier.OnUserDisconnected("user_123", true)
6. Subscriber updates internal tracking

User closes all tabs:

1. Last connection closes
2. Hub removes user from connections map
3. hasOtherConnections = false
4. Hub calls redisNotifier.OnUserDisconnected("user_123", false)
5. Subscriber removes user from subscription tracking
```

---

## Deployment Topology

### Production Setup

```
                    ┌─────────────────────────────────────┐
                    │           Load Balancer             │
                    │        (AWS ALB / Nginx)            │
                    │     wss://websocket.tantai.dev      │
                    └─────────────────┬───────────────────┘
                                      │
                    ┌─────────────────┼───────────────────┐
                    │                 │                   │
              ┌─────▼─────┐     ┌─────▼─────┐     ┌─────▼─────┐
              │ WebSocket │     │ WebSocket │     │ WebSocket │
              │ Service 1 │     │ Service 2 │     │ Service 3 │
              │  (Pod)    │     │  (Pod)    │     │  (Pod)    │
              └─────┬─────┘     └─────┬─────┘     └─────┬─────┘
                    │                 │                 │
                    └─────────────────┼─────────────────┘
                                      │
                    ┌─────────────────▼───────────────────┐
                    │         Redis Cluster               │
                    │   (ElastiCache / Redis Sentinel)    │
                    │                                     │
                    │  ┌─────────┐  ┌─────────────────┐  │
                    │  │  DB 0   │  │      DB 1       │  │
                    │  │ Pub/Sub │  │   State Store   │  │
                    │  └─────────┘  └─────────────────┘  │
                    └─────────────────────────────────────┘
```

### Kubernetes Configuration

```yaml
# k8s/websocket-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: websocket-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: websocket-service
  template:
    metadata:
      labels:
        app: websocket-service
    spec:
      containers:
        - name: websocket
          image: websocket-service:latest
          ports:
            - containerPort: 8081
          env:
            - name: REDIS_HOST
              valueFrom:
                configMapKeyRef:
                  name: websocket-config
                  key: redis-host
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /health
              port: 8081
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /health
              port: 8081
            initialDelaySeconds: 5
            periodSeconds: 10
```

### Scaling Considerations

| Metric             | Threshold      | Action      |
| ------------------ | -------------- | ----------- |
| Active Connections | > 8000 per pod | Scale up    |
| CPU Usage          | > 70%          | Scale up    |
| Memory Usage       | > 80%          | Scale up    |
| Message Latency    | > 100ms        | Investigate |

---

## Troubleshooting

### Common Issues

#### 1. WebSocket Connection Fails

**Symptoms**: Client cannot establish WebSocket connection

**Checklist**:

- [ ] JWT token valid and not expired
- [ ] Cookie `access_token` present (check browser DevTools)
- [ ] CORS origin allowed (check server logs)
- [ ] Load balancer WebSocket support enabled
- [ ] SSL certificate valid

**Debug**:

```bash
# Check WebSocket service health
curl https://websocket.tantai.dev/health | jq

# Check if subscriber is active
curl https://websocket.tantai.dev/health | jq '.subscriber'
```

#### 2. Messages Not Received

**Symptoms**: Frontend connected but not receiving updates

**Checklist**:

- [ ] Redis Pub/Sub working
- [ ] Correct channel name (`user_noti:{user_id}`)
- [ ] Project Service publishing messages
- [ ] Subscriber active (check health endpoint)

**Debug**:

```bash
# Monitor Redis Pub/Sub
redis-cli PSUBSCRIBE "user_noti:*"

# Check if messages are being published
redis-cli MONITOR | grep PUBLISH
```

#### 3. High Latency

**Symptoms**: Delay between action and notification

**Checklist**:

- [ ] Redis connection healthy
- [ ] Network latency between services
- [ ] Collector webhook throttling (5s interval)
- [ ] WebSocket buffer not full

**Debug**:

```bash
# Check Redis latency
redis-cli --latency

# Check WebSocket metrics
curl https://websocket.tantai.dev/metrics | jq
```

#### 4. Connection Drops

**Symptoms**: WebSocket disconnects frequently

**Checklist**:

- [ ] Ping/pong working (check PongWait config)
- [ ] Load balancer idle timeout (should be > PongWait)
- [ ] Network stability
- [ ] Server resources (memory, CPU)

**Debug**:

```bash
# Check connection count
curl https://websocket.tantai.dev/health | jq '.websocket'

# Check for errors in logs
kubectl logs -l app=websocket-service --tail=100 | grep -i error
```

### Monitoring Queries

**Grafana/Prometheus**:

```promql
# Active connections
websocket_active_connections

# Message throughput
rate(websocket_messages_sent_total[5m])

# Error rate
rate(websocket_messages_failed_total[5m])

# Connection rate
rate(websocket_connections_total[5m])
```

### Log Analysis

```bash
# Find connection errors
kubectl logs -l app=websocket-service | grep "connection rejected"

# Find Redis errors
kubectl logs -l app=websocket-service | grep "Redis"

# Find message routing issues
kubectl logs -l app=websocket-service | grep "Routed message"
```
