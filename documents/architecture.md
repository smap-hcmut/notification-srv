# Architecture

## System Architecture

This document provides an in-depth look at the technical architecture, component design, and implementation details of the WebSocket Notification Service.

## Component Architecture

### High-Level Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Server (Gin)                       │
│  - Health endpoint (/health)                                    │
│  - Metrics endpoint (/metrics)                                  │
│  - WebSocket upgrade endpoint (/ws)                             │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ├──────────► JWT Validator
               │             - Token parsing
               │             - Signature validation
               │             - Expiration checking
               │             - User ID extraction
               │
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                      WebSocket Handler                          │
│  - HTTP to WebSocket upgrade                                    │
│  - Authentication & authorization                               │
│  - Connection creation & registration                           │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                             Hub                                 │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  Connection Registry: map[userID][]*Connection        │      │
│  │  - Thread-safe with RWMutex                           │      │
│  │  - Supports multiple connections per user             │      │
│  └───────────────────────────────────────────────────────┘      │
│                                                                 │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  Channels:                                            │      │
│  │  - register   chan *Connection  (buffer: 100)         │      │
│  │  - unregister chan *Connection  (buffer: 100)         │      │
│  │  - broadcast  chan *BroadcastMsg (buffer: 1000)       │      │
│  └───────────────────────────────────────────────────────┘      │
│                                                                 │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  Metrics (atomic counters):                           │      │
│  │  - totalConnections                                   │      │
│  │  - totalMessagesSent                                  │      │
│  │  - totalMessagesReceived                              │      │
│  │  - totalMessagesFailed                                │      │
│  └───────────────────────────────────────────────────────┘      │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Connection                              │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  WebSocket Conn: *gorilla/websocket.Conn              │      │
│  │  User ID: string                                      │      │
│  │  Send Channel: chan []byte (buffer: 256)              │      │
│  └───────────────────────────────────────────────────────┘      │
│                                                                 │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  Read Pump (goroutine):                               │      │
│  │  - Reads messages from WebSocket                      │      │
│  │  - Handles Pong frames                                │      │
│  │  - Detects disconnections                             │      │
│  └───────────────────────────────────────────────────────┘      │
│                                                                 │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  Write Pump (goroutine):                              │      │
│  │  - Sends messages from send channel                   │      │
│  │  - Sends Ping frames every 30s                        │      │
│  │  - Handles write timeouts                             │      │
│  └───────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                       Redis Subscriber                          │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  PubSub: *redis.PubSub                                │      │
│  │  Pattern: "user_noti:*"                               │      │
│  └───────────────────────────────────────────────────────┘      │
│                                                                 │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  Listen Loop (goroutine):                             │      │
│  │  - Receives messages from Redis                       │      │
│  │  - Extracts user ID from channel name                 │      │
│  │  - Deserializes JSON payload                          │      │
│  │  - Sends to Hub for routing                           │      │
│  └───────────────────────────────────────────────────────┘      │
│                                                                 │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  Reconnection Logic:                                  │      │
│  │  - Max retries: 10                                    │      │
│  │  - Retry delay: 5 seconds                             │      │
│  │  - Automatic resubscription                           │      │
│  └───────────────────────────────────────────────────────┘      │
└──────────────┬──────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Redis Client                            │
│  - Connection pooling (size: 100, min idle: 10)                 │
│  - TLS support                                                  │
│  - Automatic retries (max: 3)                                   │
│  - Connection lifecycle management                              │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Hub (internal/websocket/hub.go)

The Hub is the central component that manages all WebSocket connections and routes messages.

#### Responsibilities
- Maintain connection registry (user ID to connections mapping)
- Handle connection registration and unregistration
- Route messages from Redis Subscriber to user connections
- Track metrics and statistics
- Coordinate graceful shutdown

#### Data Structures

```go
type Hub struct {
    // Connection registry: userID -> []*Connection
    connections map[string][]*Connection
    mu          sync.RWMutex

    // Communication channels
    register   chan *Connection
    unregister chan *Connection
    broadcast  chan *BroadcastMessage

    // Metrics (atomic for thread-safety)
    totalConnections      atomic.Int64
    totalMessagesSent     atomic.Int64
    totalMessagesReceived atomic.Int64
    totalMessagesFailed   atomic.Int64

    // Configuration
    maxConnections int

    // Lifecycle management
    ctx    context.Context
    cancel context.CancelFunc
    done   chan struct{}
}
```

#### Key Operations

**Registration Flow:**
1. Receive connection via register channel
2. Acquire write lock
3. Check if max connections limit reached
4. Add connection to user's connection list
5. Increment total connection counter
6. Release lock
7. Log connection event

**Message Routing Flow:**
1. Receive broadcast message via broadcast channel
2. Acquire read lock
3. Look up user's connections in registry
4. Release lock
5. If user offline, skip silently
6. Serialize message to JSON
7. Send to all user's connections via send channels
8. Update metrics (sent/failed counters)

**Unregistration Flow:**
1. Receive connection via unregister channel
2. Acquire write lock
3. Find and remove connection from user's list
4. Close connection's send channel
5. Decrement total connection counter
6. If last connection, remove user entry
7. Release lock
8. Log disconnection event

#### Thread Safety
- Uses sync.RWMutex for connection registry
- Read lock for lookups (GetStats, message routing)
- Write lock for modifications (register, unregister)
- Atomic counters for metrics (no lock needed)

#### Scalability Considerations
- Buffered channels prevent blocking (100 for register/unregister, 1000 for broadcast)
- Read locks allow concurrent message routing
- Map operations are O(1) for user lookup
- Connection limit prevents resource exhaustion

### 2. Connection (internal/websocket/connection.go)

Each Connection represents a single WebSocket connection to a user.

#### Responsibilities
- Manage individual WebSocket connection lifecycle
- Handle bidirectional communication (read/write)
- Implement keep-alive mechanism (Ping/Pong)
- Detect and handle disconnections
- Buffer outbound messages

#### Data Structures

```go
type Connection struct {
    hub  *Hub
    conn *websocket.Conn
    userID string
    send chan []byte  // Buffered: 256 messages

    // Configuration
    pongWait   time.Duration  // 60s
    pingPeriod time.Duration  // 30s
    writeWait  time.Duration  // 10s

    logger log.Logger
    done   chan struct{}
}
```

#### Read Pump (goroutine)

```
Start readPump() goroutine
    ↓
Set read deadline (now + pongWait)
    ↓
Set pong handler (resets read deadline)
    ↓
Set max message size limit
    ↓
Loop forever:
    ↓
    Read message from WebSocket
    ↓
    ┌─ If error:
    │   ├─ If unexpected close: log error
    │   └─ Break loop
    │
    ├─ If pong frame received:
    │   └─ Reset read deadline (handled by pong handler)
    │
    └─ If message received:
        └─ Log for debugging (we don't process client messages)
    ↓
Exit loop
    ↓
Send connection to hub.unregister channel
    ↓
Close WebSocket connection
    ↓
End goroutine
```

**Purpose:**
- Detect disconnections (network failure, client close)
- Handle Pong responses to keep connection alive
- Trigger cleanup when connection dies

#### Write Pump (goroutine)

```
Start writePump() goroutine
    ↓
Create ticker for ping interval (30s)
    ↓
Loop forever:
    ↓
    Select on:
    │
    ├─ Message from send channel:
    │   ├─ Set write deadline
    │   ├─ If channel closed: send close frame and exit
    │   ├─ Get next writer
    │   ├─ Write message
    │   ├─ Check for additional queued messages
    │   ├─ Write all queued messages in batch
    │   └─ Close writer
    │
    ├─ Ticker fires (every 30s):
    │   ├─ Set write deadline
    │   └─ Send Ping frame
    │
    └─ Done signal:
        └─ Exit loop
    ↓
Stop ticker
    ↓
Close WebSocket connection
    ↓
End goroutine
```

**Purpose:**
- Send messages from Hub to client
- Send periodic Ping frames (every 30s)
- Handle write timeouts
- Batch multiple messages for efficiency

#### Keep-Alive Mechanism

```
Timeline:
T=0s    Connection established
        ├─ Read deadline set to T+60s
        └─ Write pump starts with 30s ticker

T=30s   Write pump sends Ping
        └─ Client responds with Pong

T=30s   Pong received
        └─ Read deadline reset to T+90s

T=60s   Write pump sends Ping
        └─ Client responds with Pong

T=60s   Pong received
        └─ Read deadline reset to T+120s

...repeats...

If no Pong received:
T=90s   Read deadline exceeded
        ├─ Read pump detects timeout
        ├─ Connection unregistered
        └─ Connection closed
```

**Configuration:**
- `pongWait = 60s`: How long to wait for Pong response
- `pingPeriod = 30s`: How often to send Ping (must be < pongWait)
- `writeWait = 10s`: Maximum time for write operation

### 3. WebSocket Handler (internal/websocket/handler.go)

The Handler manages the HTTP to WebSocket upgrade process and authentication.

#### Responsibilities
- Handle HTTP WebSocket upgrade requests
- Authenticate users via JWT tokens
- Extract user identity from tokens
- Create and register new connections
- Configure CORS settings

#### Request Flow

```
Client Request: GET /ws?token=JWT_TOKEN
    ↓
Extract token from query parameter
    ↓
    ┌─ If token missing:
    │   ├─ Return 401 Unauthorized
    │   └─ End
    │
Validate JWT token
    ↓
    ┌─ If token invalid/expired:
    │   ├─ Log warning
    │   ├─ Return 401 with error message
    │   └─ End
    │
Extract user ID from token (sub claim)
    ↓
Upgrade HTTP connection to WebSocket
    ↓
    ┌─ If upgrade fails:
    │   ├─ Log error
    │   └─ End
    │
Create Connection object
    ↓
Send connection to hub.register channel
    ↓
Notify Redis Subscriber (user connected)
    ↓
Start connection read and write pumps
    ↓
Log success
    ↓
Return (connection now managed by pumps)
```

#### Authentication Details

**JWT Token Structure:**
```json
{
  "sub": "user123",           // User ID (required)
  "email": "user@example.com", // Optional
  "exp": 1800000000,          // Expiration timestamp
  "iat": 1700000000           // Issued at timestamp
}
```

**Validation Steps:**
1. Parse JWT token string
2. Verify signature using secret key
3. Check expiration time
4. Extract "sub" claim as user ID
5. Validate user ID is non-empty

**Security Considerations:**
- Token in query parameter (not header) for WebSocket compatibility
- Short-lived tokens recommended (1-24 hours)
- Secret key must be strong and rotated regularly
- No token refresh mechanism (client must reconnect)

### 4. Redis Subscriber (internal/redis/subscriber.go)

The Subscriber listens to Redis Pub/Sub channels and routes messages to the Hub.

#### Responsibilities
- Establish and maintain Redis Pub/Sub connection
- Subscribe to notification channel pattern
- Listen for published messages
- Parse and validate message format
- Route messages to Hub by user ID
- Handle Redis connection failures with retry

#### Subscription Model

**Pattern Subscription:**
```
Pattern: user_noti:*

Matches:
- user_noti:user123
- user_noti:user456
- user_noti:admin
- user_noti:guest_xyz
```

**Why Pattern Subscription:**
- No need to subscribe/unsubscribe per user
- Handles all users with single subscription
- Reduces Redis commands
- Simplifies connection management
- Supports dynamic user base

#### Message Flow

```
Backend Service
    ↓
PUBLISH user_noti:user123 '{"type":"notification","payload":{...}}'
    ↓
Redis Pub/Sub
    ↓
Subscriber's PubSub channel
    ↓
Listen loop receives message
    ↓
Extract user ID from channel name
    │  (split "user_noti:user123" → "user123")
    ↓
Parse JSON payload
    │  {
    │    "type": "notification",
    │    "payload": {...}
    │  }
    ↓
Create WebSocket Message
    │  {
    │    "type": "notification",
    │    "payload": {...},
    │    "timestamp": "2025-01-21T10:30:00Z"
    │  }
    ↓
Send to Hub.SendToUser(userID, message)
    ↓
Hub routes to user's connections
    ↓
Delivered to client
```

#### Message Format

**Redis Message (Published):**
```json
{
  "type": "notification",
  "payload": {
    "title": "New Order",
    "message": "You have a new order #12345"
  }
}
```

**WebSocket Message (Delivered):**
```json
{
  "type": "notification",
  "payload": {
    "title": "New Order",
    "message": "You have a new order #12345"
  },
  "timestamp": "2025-01-21T10:30:00Z"
}
```

#### Reconnection Strategy

```
Normal Operation:
Listen on PubSub channel
    ↓
    ┌─ Channel closed or error:
    │   ↓
    │   Log error
    │   ↓
    │   Attempt reconnection:
    │   ↓
    │   For i = 1 to maxRetries (10):
    │       ↓
    │       Close old PubSub
    │       ↓
    │       Create new PubSub
    │       ↓
    │       Subscribe to pattern
    │       ↓
    │       Test connection (Receive)
    │       ↓
    │       ┌─ If success:
    │       │   ├─ Log success
    │       │   └─ Return to normal operation
    │       │
    │       └─ If failure:
    │           ├─ Wait 5 seconds
    │           └─ Retry
    │   ↓
    │   If all retries failed:
    │   ├─ Log critical error
    │   └─ Shutdown subscriber
    │
    └─ Continue listening
```

**Reconnection Configuration:**
- Max retries: 10 attempts
- Retry delay: 5 seconds between attempts
- Total max wait: 50 seconds
- If all fail: Service should restart

### 5. HTTP Server (internal/server/)

The HTTP server provides the WebSocket endpoint and monitoring endpoints.

#### Endpoints

**WebSocket Endpoint:**
```
GET /ws?token=JWT_TOKEN
Upgrade: websocket
Connection: Upgrade
```

**Health Check Endpoint:**
```
GET /health

Response 200 OK:
{
  "status": "healthy",
  "timestamp": "2025-01-21T10:30:00Z",
  "redis": {
    "status": "connected",
    "ping_ms": 1.23
  },
  "websocket": {
    "active_connections": 1234,
    "total_unique_users": 890
  },
  "uptime_seconds": 3600
}

Response 503 Service Unavailable:
{
  "status": "degraded",
  "timestamp": "2025-01-21T10:30:00Z",
  "redis": {
    "status": "disconnected",
    "error": "connection timeout"
  },
  "websocket": {
    "active_connections": 0,
    "total_unique_users": 0
  },
  "uptime_seconds": 3600
}
```

**Metrics Endpoint:**
```
GET /metrics

Response 200 OK:
{
  "service": "websocket-service",
  "timestamp": "2025-01-21T10:30:00Z",
  "uptime_seconds": 3600,
  "connections": {
    "active": 1234,
    "total_unique_users": 890
  },
  "messages": {
    "received_from_redis": 56789,
    "sent_to_clients": 67890,
    "failed": 12
  }
}
```

## Data Flow Diagrams

### Connection Establishment Flow

```
Client                Handler              JWT Validator         Hub                  Connection
  │                     │                       │                 │                        │
  ├─ GET /ws?token=XXX ►│                       │                 │                        │
  │                     │                       │                 │                        │
  │                     ├─ Validate(token) ────►│                 │                        │
  │                     │                       │                 │                        │
  │                     │◄──── userID ──────────┤                 │                        │
  │                     │                       │                 │                        │
  │                     ├─ Upgrade to WS ───────┤                 │                        │
  │◄──── 101 Switching ─┤                       │                 │                        │
  │                     │                       │                 │                        │
  │                     ├─────── NewConnection() ────────────────────────────────────────►│
  │                     │                       │                 │                        │
  │                     ├────── register ──────────────────────►  │                        │
  │                     │                       │                 │                        │
  │                     │                       │                 ├─ Add to registry      │
  │                     │                       │                 │                        │
  │                     ├────────────────────────────────────────────── Start() ──────────►│
  │                     │                       │                 │                        │
  │                     │                       │                 │                   Start read pump
  │                     │                       │                 │                   Start write pump
  │                     │                       │                 │                        │
  │◄────────────────────────────────────────────────────────────────── Ping every 30s ────┤
  │                     │                       │                 │                        │
  ├──────────────────────────────────────────────────────────────────── Pong ────────────►│
  │                     │                       │                 │                        │
```

### Message Delivery Flow

```
Backend Service    Redis       Subscriber          Hub                 Connection         Client
      │              │             │                │                      │                 │
      ├─ PUBLISH ───►│             │                │                      │                 │
      │ user_noti:   │             │                │                      │                 │
      │ user123      │             │                │                      │                 │
      │              │             │                │                      │                 │
      │              ├─ Message ──►│                │                      │                 │
      │              │             │                │                      │                 │
      │              │             ├─ Parse channel │                      │                 │
      │              │             ├─ Extract userID│                      │                 │
      │              │             ├─ Parse JSON    │                      │                 │
      │              │             │                │                      │                 │
      │              │             ├─ SendToUser() ─►                      │                 │
      │              │             │                │                      │                 │
      │              │             │                ├─ Lookup connections  │                 │
      │              │             │                │                      │                 │
      │              │             │                ├────── send chan ────►│                 │
      │              │             │                │                      │                 │
      │              │             │                │                      ├─ Write to WS ──►│
      │              │             │                │                      │                 │
      │              │             │                │                  Update metrics       │
      │              │             │                │                      │                 │
```

### Disconnection Flow

```
Client          Connection              Hub                    Subscriber
  │                 │                    │                         │
  ├─ Close ────────►│                    │                         │
  │                 │                    │                         │
  │              Read pump               │                         │
  │              detects error           │                         │
  │                 │                    │                         │
  │                 ├─ unregister ──────►│                         │
  │                 │                    │                         │
  │                 │                 Remove from registry         │
  │                 │                    │                         │
  │                 │                    ├─ OnUserDisconnected() ─►│
  │                 │                    │                         │
  │                 │                 Close send channel           │
  │                 │                    │                         │
  │                 ├─ Close WS conn    │                    Mark as offline
  │                 │                    │                         │
  │              Stop read pump          │                         │
  │              Stop write pump         │                         │
  │                 │                    │                         │
```

## Configuration System

### Environment Variables

All configuration is loaded from environment variables using the `caarlos0/env` library.

**Server Configuration:**
```
WS_HOST=0.0.0.0             # Listen address
WS_PORT=8081                # Listen port
WS_MODE=release             # Gin mode (debug/release)
```

**Redis Configuration:**
```
REDIS_HOST=localhost        # Redis server host
REDIS_PORT=6379             # Redis server port
REDIS_PASSWORD=secret       # Redis password
REDIS_DB=0                  # Redis database number
REDIS_USE_TLS=false         # Enable TLS
REDIS_MAX_RETRIES=3         # Max retry attempts
REDIS_MIN_IDLE_CONNS=10     # Min idle connections in pool
REDIS_POOL_SIZE=100         # Max connections in pool
REDIS_POOL_TIMEOUT=4s       # Pool timeout
REDIS_CONN_MAX_IDLE_TIME=5m # Connection max idle time
REDIS_CONN_MAX_LIFETIME=30m # Connection max lifetime
```

**WebSocket Configuration:**
```
WS_PING_INTERVAL=30s        # Ping interval
WS_PONG_WAIT=60s            # Pong wait timeout
WS_WRITE_WAIT=10s           # Write operation timeout
WS_MAX_MESSAGE_SIZE=512     # Max message size in bytes
WS_READ_BUFFER_SIZE=1024    # WebSocket read buffer
WS_WRITE_BUFFER_SIZE=1024   # WebSocket write buffer
WS_MAX_CONNECTIONS=10000    # Max concurrent connections
```

**JWT Configuration:**
```
JWT_SECRET_KEY=secret       # JWT signing secret
```

**Logger Configuration:**
```
LOGGER_LEVEL=info           # Log level (debug/info/warn/error)
LOGGER_MODE=production      # Mode (development/production)
LOGGER_ENCODING=json        # Encoding (json/console)
```

**Discord Configuration (Optional):**
```
DISCORD_WEBHOOK_ID=xxx      # Discord webhook ID
DISCORD_WEBHOOK_TOKEN=xxx   # Discord webhook token
```

### Configuration Loading

```go
type Config struct {
    Server    ServerConfig
    Redis     RedisConfig
    WebSocket WebSocketConfig
    JWT       JWTConfig
    Logger    LoggerConfig
    Discord   DiscordConfig
}

func Load() (*Config, error) {
    cfg := &Config{}
    err := env.Parse(cfg)  // Parses all env vars with struct tags
    return cfg, err
}
```

## Concurrency Model

### Goroutines

The service uses goroutines extensively for concurrency:

1. **Main Goroutine**: Application lifecycle and signal handling
2. **Hub.Run()**: Hub's main loop (1 per application)
3. **Subscriber.listen()**: Redis subscription listener (1 per application)
4. **Connection.readPump()**: One per WebSocket connection
5. **Connection.writePump()**: One per WebSocket connection
6. **HTTP Server**: Multiple goroutines per HTTP request (managed by Go's http package)

**Total Goroutines** = 3 + (2 * number_of_connections)

With 10,000 connections: ~20,003 goroutines

### Synchronization

**Channels (CSP model):**
- Hub uses channels for register/unregister/broadcast
- Connection uses channel for send buffer
- Context cancellation for shutdown coordination

**Mutexes:**
- Hub uses sync.RWMutex for connection registry
- Subscriber uses sync.RWMutex for subscription tracking
- Read locks for read operations (concurrent)
- Write locks for modifications (exclusive)

**Atomic Operations:**
- Hub metrics use atomic.Int64 (no lock needed)
- Lock-free counters for performance

### Shutdown Coordination

```
Main receives SIGINT/SIGTERM
    ↓
Create shutdown context (30s timeout)
    ↓
Shutdown Subscriber:
    ├─ Cancel context
    ├─ Close PubSub connection
    └─ Wait for listen goroutine to exit
    ↓
Shutdown Hub:
    ├─ Cancel context
    ├─ Close all connections (triggers unregister)
    └─ Wait for Hub goroutine to exit
    ↓
Shutdown HTTP Server:
    ├─ Stop accepting new requests
    ├─ Wait for active requests to complete
    └─ Close listeners
    ↓
Exit application
```

## Error Handling

### Error Categories

**Authentication Errors:**
- Missing token: 401 Unauthorized
- Invalid token: 401 Unauthorized
- Expired token: 401 Unauthorized

**Connection Errors:**
- Upgrade failure: Log error, return HTTP error
- Read error: Unregister and close connection
- Write error: Unregister and close connection
- Timeout: Close connection

**Redis Errors:**
- Connection failure: Attempt reconnection
- Subscription failure: Attempt reconnection
- Parse error: Log and skip message

**System Errors:**
- Max connections: Reject new connection
- Full send buffer: Skip message, log warning
- Shutdown timeout: Force exit

### Error Recovery

**Recoverable Errors:**
- Redis disconnection: Automatic reconnection
- Single connection failure: Close that connection only
- Message parse error: Skip and continue
- Full send buffer: Drop message, continue

**Non-Recoverable Errors:**
- Redis max retries exceeded: Service shutdown
- Configuration error: Service won't start
- Listen port already in use: Service won't start

## Performance Optimizations

### Memory Optimizations
- Connection pooling for Redis (reuse connections)
- Buffered channels (reduce allocations)
- Structured logging (avoid string concatenation)
- Gorilla WebSocket (efficient message handling)
- JSON serialization caching (reuse buffers)

### CPU Optimizations
- Read locks for lookups (parallel reads)
- Atomic counters (no lock overhead)
- Pattern subscription (single Redis subscription)
- Batch message writes (reduce syscalls)
- Ticker reuse (single timer per connection)

### Network Optimizations
- Message batching in write pump
- Keep-alive reduces reconnections
- Redis connection pooling
- TLS session resumption (when enabled)

### Scalability Patterns
- Stateless design (no shared state between instances)
- Horizontal scaling (run multiple instances)
- Load balancer with sticky sessions (optional)
- Redis shared between all instances
- Independent failure domains

## Security Architecture

### Authentication Layer
```
Client ──► JWT Token ──► Handler ──► Validator ──► Extract User ID
                          │
                          ▼
                    Reject if invalid
                    (401 Unauthorized)
```

### Connection Security
- JWT validation on every connection
- No token refresh (requires reconnection)
- User ID from trusted token only
- No client-provided user ID accepted

### Network Security
- TLS for Redis (configurable)
- CORS configuration for WebSocket
- Rate limiting ready (configurable)
- Connection limits prevent DoS

### Container Security
- Non-root user (UID 65532)
- Distroless base image
- No shell in container
- Minimal attack surface
- Read-only root filesystem (optional)

## Monitoring Architecture

### Metrics Collection
```
Hub Metrics (atomic counters):
├─ totalConnections      (real-time count)
├─ totalMessagesSent     (cumulative)
├─ totalMessagesReceived (cumulative)
└─ totalMessagesFailed   (cumulative)

Connection Registry:
├─ Active connections per user
└─ Total unique users

Redis Health:
└─ Ping latency
```

### Health Checks

**Shallow Health Check:**
- HTTP server responsive
- Redis connection alive (PING)
- Basic sanity

**Deep Health Check:**
- Redis Pub/Sub working
- Message delivery working
- Connection acceptance working

### Logging Strategy

**Structured Logging (Zap):**
```json
{
  "level": "info",
  "ts": "2025-01-21T10:30:00.123Z",
  "caller": "websocket/hub.go:99",
  "msg": "User connected: user123",
  "user_id": "user123",
  "total_connections": 1234,
  "user_connections": 2
}
```

**Log Levels:**
- **DEBUG**: Detailed flow (message routing, raw data)
- **INFO**: Normal operations (connections, startup)
- **WARN**: Recoverable issues (reconnections, full buffers)
- **ERROR**: Serious errors (auth failures, Redis errors)

## Deployment Architecture

### Single Instance Deployment
```
┌─────────────────────────────────────┐
│         Load Balancer               │
│      (Optional, HTTPS termination)  │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│    WebSocket Service Instance       │
│         (Docker Container)          │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│           Redis Server              │
│        (Persistent Storage)         │
└─────────────────────────────────────┘
```

### Multi-Instance Deployment (Horizontal Scaling)
```
                ┌─────────────────────────────────────┐
                │         Load Balancer               │
                │   (Sticky sessions or any request)  │
                └─────────────┬───────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  WS Instance  │     │  WS Instance  │     │  WS Instance  │
│       1       │     │       2       │     │       3       │
└───────┬───────┘     └───────┬───────┘     └───────┬───────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
                ┌─────────────────────────────────────┐
                │           Redis Server              │
                │   (Shared message broker for all)   │
                └─────────────────────────────────────┘
```

**Scaling Characteristics:**
- Each instance handles its own connections
- Redis Pub/Sub distributes to all instances
- User may connect to any instance
- Multiple tabs may connect to different instances
- All instances receive all messages (Redis pattern subscription)
- Each instance delivers only to its own connections

### Container Deployment
```
Dockerfile (Multi-stage build):
Stage 1 (builder):
  ├─ golang:1.25 base image
  ├─ Copy go.mod, go.sum
  ├─ Download dependencies
  ├─ Copy source code
  └─ Build static binary

Stage 2 (runtime):
  ├─ gcr.io/distroless/static:nonroot
  ├─ Copy binary from builder
  ├─ Set non-root user (UID 65532)
  ├─ Expose port 8081
  └─ CMD ["/websocket-server"]

Result:
  ├─ Image size: ~20MB
  ├─ No shell, no package manager
  ├─ Minimal attack surface
  └─ Fast startup (<1s)
```

## Summary

This architecture provides:
- High concurrency through goroutines and channels
- Thread-safety through mutexes and atomic operations
- Reliability through reconnection and health checks
- Scalability through stateless design and horizontal scaling
- Security through JWT authentication and container hardening
- Observability through structured logging and metrics

The design follows Go best practices and implements all 12 requirements from the technical specification.

