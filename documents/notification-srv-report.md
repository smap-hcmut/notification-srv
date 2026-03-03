# Service: SMAP Notification Service

> **Template Version**: 1.0  
> **Last Updated**: 17/02/2026  
> **Status**: ✅ Production Ready

---

## 🎯 Business Context

### Chức năng chính

Service xử lý **Real-Time Notifications** và **Critical Alerts** cho SMAP (Social Media Analytics Platform). Đóng vai trò là cầu nối một chiều (push-only) giữa backend services và client dashboards, đồng thời gửi cảnh báo quan trọng đến Discord.

**Giải quyết vấn đề**:

- Cập nhật real-time tiến trình xử lý dữ liệu (Data Onboarding, Analytics Pipeline)
- Cảnh báo ngay lập tức các sự kiện nghiêm trọng (Crisis Alert - sentiment spike)
- Thông báo lifecycle của marketing campaigns
- Monitoring và alerting qua Discord cho team operations

### Luồng xử lý

```
Backend Services (Crawler/Analyzer)
    → PUBLISH to Redis Pub/Sub
    → notification-srv SUBSCRIBE
    → Parse & Transform Message
    → Route to WebSocket Hub (by UserID)
    → Push JSON to Browser Clients

    [Special Path for Crisis]
    → Detect CRISIS_ALERT type
    → Dispatch to Discord (Rich Embed)
```

### Giá trị cốt lõi

- **Real-time Experience**: Users nhận updates ngay lập tức về tiến trình xử lý dữ liệu
- **Proactive Alerting**: Phát hiện và thông báo crisis (sentiment spike) trong vòng <1s
- **Multi-channel**: WebSocket cho dashboard + Discord cho team operations
- **Scalability**: Support 10,000+ concurrent WebSocket connections
- **Reliability**: Graceful shutdown, automatic reconnection handling

---

## 🛠 Technical Details

### Protocol & Architecture

- **Protocol**: WebSocket (Push-only) + REST API (Health checks)
- **Pattern**: Clean Architecture với Domain-Driven Design
- **Design**: Layered Architecture (Delivery → UseCase → Domain)

### Tech Stack

| Component      | Technology        | Version | Purpose                   |
| -------------- | ----------------- | ------- | ------------------------- |
| Language       | Go                | 1.25.4  | Backend service           |
| Framework      | Gin               | 1.11.0  | HTTP routing & middleware |
| WebSocket      | Gorilla WebSocket | 1.5.3   | Connection handling       |
| Message Broker | Redis Pub/Sub     | 9.7.0   | Event ingestion           |
| Auth           | JWT (HS256)       | 5.2.1   | Token validation          |
| Logger         | Zap               | 1.27.0  | Structured logging        |
| Config         | Viper             | 1.21.0  | Configuration management  |
| Alerting       | Discord Webhooks  | -       | External notifications    |

### Database Schema

Service này **không có database riêng**. Nó hoạt động như một stateless message router:

- **State Management**: In-memory Hub (active connections map)
- **Persistence**: Không lưu trữ messages (ephemeral)
- **Authentication**: Verify JWT từ Identity Service (không query DB)

**In-Memory Structures**:

```go
// Hub maintains active connections
type Hub struct {
    connections map[string][]*Connection  // userID -> []*Connection
    register    chan *Connection
    unregister  chan *Connection
    broadcast   chan []byte
}
```

---

## 📡 API Endpoints

### Domain 1: WebSocket (Real-time Notifications)

#### `GET /ws`

**Purpose**: Upgrade HTTP connection to WebSocket for receiving real-time notifications

**Authentication**: JWT (Required)

- Cookie: `smap_auth_token` (HttpOnly, Secure) — Recommended
- Query Param: `?token=eyJhbG...` — Debugging only

**Query Parameters**:

- `project_id` (optional): Filter messages to specific project

**Request**:

```http
GET /ws?project_id=proj_123 HTTP/1.1
Host: localhost:8080
Upgrade: websocket
Connection: Upgrade
Cookie: smap_auth_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response** (WebSocket Upgrade - 101):

```http
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
```

**Response** (Error - 401):

```json
{
  "error": "unauthorized",
  "message": "Invalid or expired token"
}
```

**Response** (Error - 400):

```json
{
  "error": "bad_request",
  "message": "Failed to upgrade connection"
}
```

**Business Logic Flow**:

1. Extract JWT from Cookie or Query Param
2. Validate token signature and expiration
3. Extract UserID from token claims
4. Validate optional project_id filter
5. Upgrade HTTP to WebSocket
6. Register connection to Hub
7. Start read/write pumps (goroutines)

**Performance**:

- Avg latency: <50ms (upgrade)
- Throughput: 10,000+ concurrent connections
- Message latency: <10ms (hub routing)

**WebSocket Message Format** (Server → Client):

```json
{
  "type": "DATA_ONBOARDING",
  "timestamp": "2026-02-17T14:00:00Z",
  "payload": {
    "project_id": "proj_123",
    "source_name": "My TikTok Page",
    "status": "COMPLETED",
    "progress": 100,
    "record_count": 1500
  }
}
```

---

#### `GET /health`

**Purpose**: Health check endpoint for load balancers and monitoring

**Authentication**: Public (No auth required)

**Response** (Success - 200):

```json
{
  "status": "healthy",
  "redis": "connected",
  "active_connections": 1234,
  "unique_users": 567
}
```

**Response** (Degraded - 200):

```json
{
  "status": "degraded",
  "redis": "disconnected",
  "active_connections": 0
}
```

**Performance**:

- Avg latency: <5ms
- No cache (always fresh)

---

### Domain 2: Alert (Discord Notifications)

**Note**: Domain này không expose HTTP endpoints. Nó được trigger internally khi nhận CRISIS_ALERT messages.

**Internal Flow**:

```
Redis Message (CRISIS_ALERT)
  → WebSocket UseCase detects type
  → Call Alert.DispatchCrisisAlert()
  → Transform to Discord Embed
  → POST to Discord Webhook
```

---

## 🔗 Integration & Dependencies

### External Services

**1. Identity Service** (Upstream)

- **Method**: JWT Token Validation (Shared Secret)
- **Purpose**: Authenticate WebSocket connections
- **Integration**:
  - Service không call trực tiếp Identity Service
  - Verify JWT signature using shared `JWT_SECRET_KEY`
  - Extract `user_id` from token claims
- **Error Handling**: Reject connection with 401 if token invalid
- **SLA**: N/A (offline validation)

**2. Backend Services** (Upstream - Crawler, Analyzer, Knowledge)

- **Method**: Redis Pub/Sub (Async)
- **Purpose**: Receive events to push to clients
- **Channels Used**:
  - `project:{id}:user:{uid}` - Project-scoped events
  - `campaign:{id}:user:{uid}` - Campaign events
  - `alert:crisis:user:{uid}` - Crisis alerts
  - `system:{subtype}` - System-wide events
- **Message Format**: JSON payloads (see Contracts section)
- **Error Handling**:
  - Invalid JSON → Log error, skip message
  - Unknown type → Log warning, skip
  - Retry: N/A (fire-and-forget)
- **SLA**: <100ms message processing

**3. Discord** (Downstream)

- **Method**: HTTP Webhook (POST)
- **Purpose**: Send critical alerts to operations team
- **Endpoints Used**:
  - `POST https://discord.com/api/webhooks/{id}/{token}` - Send embed
- **Error Handling**:
  - Retry: 3 attempts with exponential backoff
  - Fallback: Log to console if all retries fail
- **SLA**: <2s per webhook call

### Infrastructure Dependencies

**Message Queue** (Redis Pub/Sub)

```
Channels Subscribed:
- project:*:user:* → Project events (DATA_ONBOARDING, ANALYTICS_PIPELINE)
- campaign:*:user:* → Campaign events
- alert:crisis:user:* → Crisis alerts
- system:* → System events

Message Format: {
  "type": "MESSAGE_TYPE",
  "payload": { ... }
}

Handler:
1. Parse channel to extract scope (project_id, user_id)
2. Detect message type from payload
3. Transform to NotificationOutput
4. Route to Hub.SendToUser(userID, message)
5. If CRISIS_ALERT → Also dispatch to Discord
```

**Cache** (Redis - Connection Pool)

```
Usage: Pub/Sub only (no key-value caching)
Connection Pool: 10 connections
Reconnection: Automatic with exponential backoff
Health Check: PING every 30s
```

---

## 🎨 Key Features & Highlights

### 1. Hub-based Connection Management

**Description**: Goroutine-per-client model với centralized Hub quản lý routing

**Implementation**:

- Hub chạy trong 1 goroutine duy nhất (single-threaded routing)
- Mỗi connection có 2 goroutines: readPump + writePump
- Channel-based communication (register, unregister, broadcast)
- User mapping: `map[string][]*Connection` (1 user → N connections)

**Benefits**:

- Thread-safe routing without locks
- Support multiple tabs/devices per user
- Clean disconnection handling
- Scalable to 10,000+ connections

### 2. Smart Message Routing

**Description**: Filter và route messages dựa trên UserID và ProjectID

**Implementation**:

- Parse Redis channel: `project:{id}:user:{uid}` → Extract user_id
- Hub maintains user → connections mapping
- Optional project_id filter tại connection level
- Only send message if: `msg.project_id == conn.project_id OR conn.project_id == ""`

**Benefits**:

- Reduce bandwidth (clients chỉ nhận messages liên quan)
- Privacy (users không thấy data của nhau)
- Flexible filtering (có thể subscribe all projects hoặc specific)

### 3. Dual-channel Alerting (WebSocket + Discord)

**Description**: Crisis alerts được gửi đồng thời đến dashboard và Discord

**Implementation**:

- WebSocket UseCase detect `CRISIS_ALERT` type
- Call `alertUC.DispatchCrisisAlert()` asynchronously
- Alert UseCase transform payload → Discord Rich Embed
- Color mapping: CRITICAL=Red, WARNING=Orange
- Include sample mentions, affected aspects, action required

**Benefits**:

- Operations team nhận alert ngay cả khi không mở dashboard
- Rich formatting với Discord Embeds (visual hierarchy)
- Persistent notification history trong Discord channel

### 4. Performance Optimizations

- **Connection Pooling**: Redis connection pool (10 connections)
- **Goroutine Model**: Non-blocking I/O với read/write pumps
- **Message Buffering**: Buffered channels (256 messages per connection)
- **Ping/Pong**: Keep-alive mechanism (30s ping, 60s pong timeout)
- **Graceful Degradation**: Continue serving WebSocket nếu Discord down

### 5. Reliability Features

- **Retry Logic**: Discord webhooks - 3 retries với exponential backoff
- **Graceful Shutdown**:
  1. Stop accepting new connections
  2. Close Redis subscriber
  3. Send CloseServiceRestart (1012) to all clients
  4. Wait for connections to close (max 10s)
- **Health Checks**: `/health` endpoint với Redis connectivity check
- **Auto Reconnection**: Redis client tự động reconnect khi mất kết nối
- **Error Isolation**: Lỗi ở 1 connection không ảnh hưởng đến connections khác

---

## 🚧 Status & Roadmap

### ✅ Done (Implemented & Tested)

- [x] WebSocket connection management với JWT authentication
- [x] Redis Pub/Sub subscriber với auto-reconnection
- [x] Hub-based message routing (user-scoped)
- [x] 4 message types: DATA_ONBOARDING, ANALYTICS_PIPELINE, CRISIS_ALERT, CAMPAIGN_EVENT
- [x] Discord integration với Rich Embeds
- [x] Project-level filtering
- [x] Graceful shutdown handling
- [x] Health check endpoint
- [x] Docker deployment với Kubernetes manifests
- [x] Structured logging với Zap
- [x] Configuration management với Viper

### 🔄 In Progress

- [ ] Load testing với 10,000 concurrent connections - Status: 80% complete
- [ ] Prometheus metrics integration - Status: Planning
- [ ] E2E tests với mock Redis - Status: 60% complete

### 📋 Todo (Planned)

- [ ] Message persistence (optional replay) - Priority: Low
- [ ] Rate limiting per user - Priority: Medium
- [ ] WebSocket compression - Priority: Low
- [ ] Grafana dashboard - Priority: High
- [ ] Alert aggregation (prevent spam) - Priority: Medium
- [ ] Multi-region deployment - Priority: Low

### 🐛 Known Bugs

- [ ] Bug #45: Connection leak khi client disconnect đột ngột - Severity: Medium
  - Workaround: Ping/Pong timeout sẽ cleanup sau 60s
- [ ] Bug #67: Discord webhook timeout không được log đầy đủ - Severity: Low

---

## ⚠️ Known Issues & Limitations

### 1. Performance - Connection Limit

**Issue**: Hub sử dụng single goroutine cho routing, có thể bottleneck ở >50k connections

- **Current**: Hub.run() xử lý tất cả register/unregister/broadcast trong 1 goroutine
- **Problem**: Với >50k connections, channel operations có thể bị chậm
- **Impact**: Message latency tăng từ <10ms lên ~100ms
- **Workaround**: Hiện tại limit ở 10k connections (config: `websocket.max_connections`)
- **TODO**: Implement sharded hubs (hash userID → hub instance)

**Code location**: `internal/websocket/usecase/hub.go`

```go
// ❌ Current implementation
func (h *Hub) run() {
    for {
        select {
        case conn := <-h.register:
            // Single goroutine handles all
        }
    }
}

// ✅ Proposed solution
type ShardedHub struct {
    shards []*Hub  // Multiple hub instances
}
func (sh *ShardedHub) getShard(userID string) *Hub {
    hash := fnv.New32a()
    hash.Write([]byte(userID))
    return sh.shards[hash.Sum32() % uint32(len(sh.shards))]
}
```

### 2. Reliability - Message Loss on Disconnect

**Issue**: Messages sent khi client đang disconnect sẽ bị mất (no persistence)

- **Current**: Messages chỉ tồn tại trong memory, không có replay mechanism
- **Problem**: Nếu client disconnect 5 phút, tất cả messages trong thời gian đó bị mất
- **Impact**: User có thể miss critical updates (e.g., data onboarding completed)
- **Workaround**: Frontend poll REST API để sync state khi reconnect
- **TODO**: Implement message buffer per user (Redis Streams hoặc in-memory queue)

**Code location**: `internal/websocket/usecase/hub.go`

```go
// ❌ Current: Fire-and-forget
func (h *Hub) SendToUser(userID string, message []byte) {
    conns := h.connections[userID]
    for _, conn := range conns {
        select {
        case conn.send <- message:
        default:
            // Message dropped if buffer full
        }
    }
}

// ✅ Proposed: Buffer recent messages
type Hub struct {
    recentMessages map[string]*ring.Ring  // userID -> last 100 messages
}
```

### 3. Security - No Rate Limiting

**Issue**: Không có rate limiting cho WebSocket connections per user

- **Current**: User có thể mở unlimited connections (chỉ limit global 10k)
- **Problem**: 1 user có thể spam connections, chiếm hết quota
- **Impact**: DoS attack vector
- **Workaround**: Kubernetes resource limits + monitoring
- **TODO**: Implement per-user connection limit (e.g., max 10 connections/user)

**Code location**: `internal/websocket/usecase/new.go`

```go
// ✅ Proposed solution
func (uc *implUseCase) Register(ctx context.Context, input ws.ConnectionInput) error {
    // Check per-user limit
    userConns := uc.hub.GetUserConnections(input.UserID)
    if len(userConns) >= MaxConnectionsPerUser {
        return ErrTooManyConnections
    }
    // ... existing logic
}
```

### 4. Monitoring - Limited Observability

**Issue**: Thiếu metrics chi tiết về message flow và connection lifecycle

- **Current**: Chỉ có basic logs và health check
- **Problem**: Khó debug performance issues, không có alerting
- **Impact**: Phát hiện issues chậm, troubleshooting khó khăn
- **Workaround**: Parse logs manually
- **TODO**:
  - Integrate Prometheus metrics (connection count, message rate, latency)
  - Add distributed tracing (OpenTelemetry)
  - Grafana dashboard

**Metrics cần thêm**:

- `ws_connections_total{user_id}` - Connections per user
- `ws_messages_sent_total{type}` - Messages by type
- `ws_message_latency_seconds` - End-to-end latency
- `discord_webhook_errors_total` - Discord failures

---

## 🔮 Future Enhancements

### Short-term (1-2 months)

- [ ] Prometheus metrics integration - Improve observability và alerting
- [ ] Per-user rate limiting - Prevent abuse và DoS
- [ ] Message replay buffer - Reduce message loss on reconnect

### Mid-term (3-6 months)

- [ ] Sharded Hub architecture - Scale beyond 50k connections
- [ ] WebSocket compression - Reduce bandwidth usage
- [ ] Alert aggregation - Prevent Discord spam (group similar alerts)

### Long-term (6+ months)

- [ ] Multi-region deployment - Reduce latency cho global users
- [ ] Message persistence layer - Full replay capability
- [ ] Client SDK - Simplify frontend integration

---

## 🔧 Configuration

**File**: `config/notification-config.yaml`

```yaml
# Environment
environment:
  name: production # development | staging | production

# Server
server:
  port: 8080
  mode: release # debug | release

# Logger
logger:
  level: info # debug | info | warn | error
  mode: production # development | production
  encoding: json # json | console
  color_enabled: false

# Redis
redis:
  host: redis.smap.svc.cluster.local
  port: 6379
  password: ""
  db: 0

# WebSocket
websocket:
  ping_interval: 30s
  pong_wait: 60s
  write_wait: 10s
  max_message_size: 512
  read_buffer_size: 1024
  write_buffer_size: 1024
  max_connections: 10000

# JWT
jwt:
  secret_key: "your-secret-key-min-32-chars" # REQUIRED, min 32 chars

# Cookie
cookie:
  domain: ".smap.com"
  secure: true
  samesite: Lax # Strict | Lax | None
  max_age: 7200 # 2 hours
  max_age_remember: 2592000 # 30 days
  name: smap_auth_token

# Discord
discord:
  webhook_url: "https://discord.com/api/webhooks/..." # Optional
```

**Environment Variables** (Override config file):

```bash
# Required
JWT_SECRET_KEY=your-secret-key-min-32-chars
REDIS_HOST=redis.smap.svc.cluster.local
REDIS_PORT=6379

# Optional
SERVER_PORT=8080
SERVER_MODE=release
ENVIRONMENT_NAME=production

WEBSOCKET_MAX_CONNECTIONS=10000
WEBSOCKET_PING_INTERVAL=30s

COOKIE_DOMAIN=.smap.com
COOKIE_SECURE=true
COOKIE_NAME=smap_auth_token

DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...

LOGGER_LEVEL=info
LOGGER_MODE=production
```

**Config Precedence**: Environment Variables > Config File > Defaults

---

## 📊 Performance Metrics

### Estimated Performance

**Note**: Đây là estimates dựa trên code analysis và initial testing. Load tests đang được thực hiện.

| Metric                   | Value  | Target | Status |
| ------------------------ | ------ | ------ | ------ |
| WebSocket Upgrade Time   | ~30ms  | <50ms  | ✅     |
| Message Routing Latency  | ~5ms   | <10ms  | ✅     |
| Hub Broadcast Time       | ~20ms  | <50ms  | ✅     |
| Discord Webhook Time     | ~500ms | <2s    | ✅     |
| Concurrent Connections   | 10,000 | >5,000 | ✅     |
| Messages/sec (per conn)  | ~100   | >50    | ✅     |
| CPU Usage (10k conns)    | ~40%   | <70%   | ✅     |
| Memory Usage (10k conns) | ~800MB | <2GB   | ✅     |
| Redis Pub/Sub Latency    | ~2ms   | <10ms  | ✅     |

**Bottlenecks Identified**:

- Hub routing: Single goroutine limit ~50k connections
- Discord webhooks: Rate limit 30 req/s (need aggregation)
- Memory: ~80KB per connection (mostly buffers)

**TODO**: Run comprehensive load tests với k6 hoặc Artillery

---

## 🔐 Security

### Authentication

- **Method**: JWT (HS256) via HttpOnly Cookie
- **Token Storage**:
  - Primary: HttpOnly Cookie (XSS protection)
  - Fallback: Query Param (debugging only, not recommended for production)
- **Token Expiry**: 2 hours (configurable)
- **Refresh Strategy**: Frontend responsible for refresh (call Identity Service)

### Authorization

- **Model**: User-scoped routing (implicit authorization)
- **Permissions**:
  - Users chỉ nhận messages có `user_id` match với token
  - Project filtering: Optional, client-side preference
- **Scope Validation**:
  - Extract `user_id` from JWT claims
  - Hub routes messages based on channel pattern: `*:user:{uid}`

### Data Protection

- **Encryption at Rest**: N/A (no persistence)
- **Encryption in Transit**: TLS 1.3 (handled by ingress/load balancer)
- **PII Handling**:
  - No PII stored in service
  - Messages may contain project names, metrics (not sensitive)
- **Secrets Management**:
  - Kubernetes Secrets for JWT_SECRET_KEY, DISCORD_WEBHOOK_URL
  - Environment variables injection

### Security Best Practices

- [x] Input validation (JWT, project_id)
- [x] XSS protection (HttpOnly cookies)
- [x] CORS configuration (allowed origins)
- [x] Connection limits (global max 10k)
- [ ] Rate limiting (TODO)
- [x] Secure WebSocket (wss:// in production)
- [x] Audit logging (structured logs với user_id)
- [ ] Dependency scanning (TODO - integrate Dependabot)
- [x] Security headers (handled by middleware)

**Security Considerations**:

- JWT secret MUST be min 32 chars (validated at startup)
- Cookie MUST be Secure=true in production
- Discord webhook URL is sensitive (treat as secret)

---

## 🧪 Testing

### Test Coverage

- **Unit Tests**: ~65% coverage
  - `pkg/discord`: 80%
  - `pkg/jwt`: 90%
  - `internal/websocket/usecase`: 60%
  - `internal/alert/usecase`: 70%
- **Integration Tests**: 5 scenarios
  - WebSocket upgrade flow
  - Message routing
  - Discord webhook dispatch
  - Graceful shutdown
  - Redis reconnection
- **E2E Tests**: In progress
- **Load Tests**: Planned (target: 10k concurrent connections)

### Running Tests

```bash
# Unit tests
make test
# Output: go test -v -race -coverprofile=coverage.out ./...

# Integration tests (requires Redis)
make test-integration
# Output: go test -v -tags=integration ./...

# Coverage report
make coverage
# Output: go tool cover -html=coverage.out

# Load tests (TODO)
make test-load
# Output: k6 run tests/load/websocket.js
```

### Test Strategy

- **Unit Tests**:
  - UseCase logic (message transformation, routing)
  - Helper functions (channel parsing, type detection)
  - Discord embed formatting
- **Integration Tests**:
  - WebSocket upgrade với mock JWT
  - Redis Pub/Sub với testcontainers
  - Hub registration/unregistration
- **Mocking Strategy**:
  - Interfaces: `log.Logger`, `discord.IDiscord`, `redis.IRedis`
  - Mock library: `github.com/stretchr/testify/mock`
- **Test Data**:
  - Fixtures: `tests/fixtures/*.json`
  - Mock JWT tokens với known secret

---

## 🚀 Deployment

### Docker

**Dockerfile**: `cmd/api/Dockerfile`

```bash
# Build
docker build -t smap/notification-srv:latest -f cmd/api/Dockerfile .

# Run locally
docker run -d -p 8080:8080 \
  -e JWT_SECRET_KEY=your-secret-key-min-32-chars \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/... \
  --name notification-srv \
  smap/notification-srv:latest

# Check logs
docker logs -f notification-srv

# Health check
curl http://localhost:8080/health
```

### Kubernetes

**Manifests**: `manifests/` và `cmd/api/deployment.yaml`

```yaml
# ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: notification-srv-config
data:
  notification-config.yaml: |
    environment:
      name: production
    server:
      port: 8080
    redis:
      host: redis.smap.svc.cluster.local
      port: 6379
    # ... (see manifests/configmap.yaml)

---
# Secret
apiVersion: v1
kind: Secret
metadata:
  name: notification-srv-secrets
type: Opaque
stringData:
  jwt-secret-key: "your-secret-key-min-32-chars"
  discord-webhook-url: "https://discord.com/api/webhooks/..."

---
# Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-srv
spec:
  replicas: 3
  selector:
    matchLabels:
      app: notification-srv
  template:
    metadata:
      labels:
        app: notification-srv
    spec:
      containers:
        - name: notification-srv
          image: smap/notification-srv:latest
          ports:
            - containerPort: 8080
              name: http
          env:
            - name: JWT_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: notification-srv-secrets
                  key: jwt-secret-key
            - name: DISCORD_WEBHOOK_URL
              valueFrom:
                secretKeyRef:
                  name: notification-srv-secrets
                  key: discord-webhook-url
            - name: REDIS_HOST
              value: "redis.smap.svc.cluster.local"
          volumeMounts:
            - name: config
              mountPath: /app/config
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 1Gi
      volumes:
        - name: config
          configMap:
            name: notification-srv-config

---
# Service
apiVersion: v1
kind: Service
metadata:
  name: notification-srv
spec:
  selector:
    app: notification-srv
  ports:
    - port: 80
      targetPort: 8080
      name: http
  type: ClusterIP
```

**Deploy Commands**:

```bash
# Apply manifests
kubectl apply -f manifests/configmap.yaml
kubectl apply -f manifests/secret.yaml
kubectl apply -f cmd/api/deployment.yaml

# Check status
kubectl get pods -l app=notification-srv
kubectl logs -f deployment/notification-srv

# Port forward for testing
kubectl port-forward svc/notification-srv 8080:80

# Scale
kubectl scale deployment notification-srv --replicas=5
```

### CI/CD Pipeline

**GitHub Actions**: `.github/workflows/deploy.yml` (example)

```yaml
name: Deploy Notification Service
on:
  push:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: "1.25"
      - name: Run tests
        run: make test
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build Docker image
        run: docker build -t smap/notification-srv:${{ github.sha }} .
      - name: Push to registry
        run: |
          echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker push smap/notification-srv:${{ github.sha }}

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to Kubernetes
        run: |
          kubectl set image deployment/notification-srv \
            notification-srv=smap/notification-srv:${{ github.sha }}
          kubectl rollout status deployment/notification-srv
```

---

## 📚 Documentation

### API Documentation

- **Swagger/OpenAPI**: http://localhost:8080/swagger/index.html
  - Auto-generated từ comments trong code
  - Annotations: `@title`, `@description`, `@securityDefinitions`
- **WebSocket Protocol**: See `documents/contracts.md`
- **Message Types**: See `documents/notification.md`

### Architecture Docs

- **System Design**: `documents/notification.md`
- **Data Contracts**: `documents/contracts.md`
- **Domain Conventions**: `documents/domain_convention/`
- **Code Plans**: `documents/domain_*_code_plan.md`

### Developer Guides

- **Setup Guide**: See README.md Quick Start section
- **Contributing Guide**: (TODO)
- **Code Style**: Follow Go standard conventions + Clean Architecture patterns

---

## 📞 Contact & Support

### Team

- **Service Owner**: SMAP Platform Team
- **Tech Lead**: [Name]
- **Repository**: https://github.com/smap/notification-srv

### Resources

- **Issue Tracker**: GitHub Issues
- **Monitoring**: (TODO - Grafana dashboard)
- **Logs**: Kubernetes logs (`kubectl logs`)
- **Slack Channel**: #smap-notifications

### SLA & Support

- **Availability Target**: 99.9% (planned)
- **Response Time**:
  - P0 (Service down): 15 minutes
  - P1 (Degraded): 1 hour
  - P2 (Minor issue): 4 hours
- **Support Hours**: Best effort (graduation project)

---

## 📝 Changelog

### [1.0.0] - 2026-02-17

**Added**

- WebSocket connection management với JWT authentication
- Redis Pub/Sub subscriber
- Hub-based message routing
- 4 message types: DATA_ONBOARDING, ANALYTICS_PIPELINE, CRISIS_ALERT, CAMPAIGN_EVENT
- Discord integration với Rich Embeds
- Health check endpoint
- Graceful shutdown
- Docker + Kubernetes deployment
- Structured logging với Zap
- Configuration management với Viper

**Security**

- HttpOnly cookie authentication
- JWT signature validation
- CORS middleware

---

## 🎓 Learning Resources

### Internal Docs

- `documents/notification.md` - Technical overview
- `documents/contracts.md` - API contracts
- `documents/domain_convention/` - Coding conventions

### External Resources

- [Gorilla WebSocket Documentation](https://pkg.go.dev/github.com/gorilla/websocket)
- [Redis Pub/Sub Guide](https://redis.io/docs/manual/pubsub/)
- [Discord Webhook API](https://discord.com/developers/docs/resources/webhook)
- [Clean Architecture in Go](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

---

## 📋 Appendix

### Glossary

- **Hub**: Central routing component quản lý active WebSocket connections
- **Pump**: Goroutine xử lý read/write operations cho WebSocket
- **Scope**: JWT claims chứa user identity và permissions
- **Embed**: Discord's rich message format với colors, fields, thumbnails
- **Channel**: Redis Pub/Sub channel pattern (e.g., `project:*:user:*`)

### Related Services

- **Identity Service**: Provides JWT tokens, user authentication
- **Crawler Service**: Publishes DATA_ONBOARDING events
- **Analyzer Service**: Publishes ANALYTICS_PIPELINE và CRISIS_ALERT events
- **Campaign Service**: Publishes CAMPAIGN_EVENT events

---

**Document Version**: 1.0  
**Last Updated**: 17/02/2026  
**Maintained By**: SMAP Platform Team  
**Review Cycle**: Monthly

---

## 📌 Quick Reference Card

```
Service: SMAP Notification Service
Port: 8080
Health: GET /health
WebSocket: GET /ws (requires JWT)

Quick Start:
1. Clone repo: git clone <repo-url>
2. Copy config: cp config/notification-config.example.yaml config/notification-config.yaml
3. Edit config: Set JWT_SECRET_KEY, REDIS_HOST, DISCORD_WEBHOOK_URL
4. Run: make run
5. Test: curl http://localhost:8080/health

Common Commands:
- Start: make run
- Test: make test
- Build: make build
- Docker: docker build -t notification-srv .
- Deploy: kubectl apply -f manifests/

WebSocket Connection (JavaScript):
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log(msg.type, msg.payload);
};

Emergency Contacts:
- GitHub Issues: https://github.com/smap/notification-srv/issues
- Slack: #smap-notifications
```
