# API Reference

## API Overview

This document provides complete API documentation for the WebSocket Notification Service, including all endpoints, message formats, error codes, and usage examples.

## Base URL

```
Development:  ws://localhost:8081
Production:   wss://your-domain.com
```

## Authentication

All WebSocket connections require JWT authentication via query parameter.

### JWT Token Format

```json
{
  "sub": "user123",           // Required: User ID
  "email": "user@example.com", // Optional
  "exp": 1800000000,          // Required: Expiration timestamp (Unix)
  "iat": 1700000000           // Required: Issued at timestamp (Unix)
}
```

### Token Requirements

- Algorithm: HS256
- Claims: `sub` (user ID) is required
- Expiration: Must include valid `exp` claim
- Signature: Must match configured JWT_SECRET_KEY

## Endpoints

### 1. WebSocket Connection

Establishes a persistent WebSocket connection for real-time notifications.

#### Endpoint

```
GET /ws?token={JWT_TOKEN}
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Version: 13
```

#### Query Parameters

| Parameter | Type   | Required | Description                    |
|-----------|--------|----------|--------------------------------|
| token     | string | Yes      | JWT authentication token       |

#### Success Response (101 Switching Protocols)

```
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: {hash}
```

Connection is now established. Client will receive messages via WebSocket protocol.

#### Error Responses

**Missing Token (401 Unauthorized)**
```json
{
  "error": "missing token parameter"
}
```

**Invalid Token (401 Unauthorized)**
```json
{
  "error": "invalid or expired token"
}
```

**Connection Limit Reached (503 Service Unavailable)**
```
Connection rejected (logged server-side, no response sent)
```

#### Client Examples

**JavaScript (Browser)**
```javascript
const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';
const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received message:', message);
  
  // Handle different message types
  switch(message.type) {
    case 'notification':
      showNotification(message.payload);
      break;
    case 'alert':
      showAlert(message.payload);
      break;
    case 'update':
      handleUpdate(message.payload);
      break;
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = (event) => {
  console.log('WebSocket closed:', event.code, event.reason);
  // Implement reconnection logic here
};
```

**Go Client**
```go
package main

import (
    "fmt"
    "log"
    "github.com/gorilla/websocket"
)

func main() {
    token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    url := fmt.Sprintf("ws://localhost:8081/ws?token=%s", token)
    
    conn, _, err := websocket.DefaultDialer.Dial(url, nil)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer conn.Close()
    
    fmt.Println("WebSocket connected")
    
    for {
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Println("Read error:", err)
            break
        }
        
        fmt.Printf("Received: %s\n", message)
    }
}
```

**Python Client**
```python
import websocket
import json

def on_message(ws, message):
    data = json.loads(message)
    print(f"Received: {data}")
    
    if data['type'] == 'notification':
        print(f"Notification: {data['payload']}")

def on_error(ws, error):
    print(f"Error: {error}")

def on_close(ws, close_status_code, close_msg):
    print("WebSocket closed")

def on_open(ws):
    print("WebSocket connected")

token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
url = f"ws://localhost:8081/ws?token={token}"

ws = websocket.WebSocketApp(url,
    on_open=on_open,
    on_message=on_message,
    on_error=on_error,
    on_close=on_close)

ws.run_forever()
```

**Node.js Client**
```javascript
const WebSocket = require('ws');

const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';
const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);

ws.on('open', () => {
  console.log('WebSocket connected');
});

ws.on('message', (data) => {
  const message = JSON.parse(data);
  console.log('Received message:', message);
});

ws.on('error', (error) => {
  console.error('WebSocket error:', error);
});

ws.on('close', () => {
  console.log('WebSocket closed');
});
```

#### Connection Lifecycle

**Keep-Alive (Ping/Pong)**

The server automatically sends Ping frames every 30 seconds. Clients should respond with Pong frames (most WebSocket libraries handle this automatically).

```
T=0s    Connection established
T=30s   Server sends Ping
        Client responds with Pong
T=60s   Server sends Ping
        Client responds with Pong
...

If no Pong received within 60s:
        Server closes connection
```

**Disconnection**

The connection will be closed if:
- Client closes connection
- Network failure
- Authentication token expires (requires reconnection with new token)
- No Pong response within 60 seconds
- Server shutdown
- Connection limit reached

### 2. Health Check

Returns the health status of the service.

#### Endpoint

```
GET /health
```

#### Success Response (200 OK)

**Healthy Status:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-21T10:30:00.123Z",
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
```

#### Degraded Response (503 Service Unavailable)

**Redis Disconnected:**
```json
{
  "status": "degraded",
  "timestamp": "2025-01-21T10:30:00.123Z",
  "redis": {
    "status": "disconnected",
    "error": "connection refused"
  },
  "websocket": {
    "active_connections": 0,
    "total_unique_users": 0
  },
  "uptime_seconds": 3600
}
```

#### Response Fields

| Field                              | Type   | Description                                    |
|------------------------------------|--------|------------------------------------------------|
| status                             | string | Overall status: "healthy" or "degraded"        |
| timestamp                          | string | ISO 8601 timestamp                             |
| redis.status                       | string | Redis connection status                        |
| redis.ping_ms                      | float  | Redis ping latency in milliseconds             |
| redis.error                        | string | Error message if Redis is disconnected         |
| websocket.active_connections       | int    | Total active WebSocket connections             |
| websocket.total_unique_users       | int    | Number of unique users connected               |
| uptime_seconds                     | int    | Service uptime in seconds                      |

#### Usage Examples

**cURL**
```bash
curl http://localhost:8081/health
```

**JavaScript (Fetch)**
```javascript
fetch('http://localhost:8081/health')
  .then(response => response.json())
  .then(data => {
    if (data.status === 'healthy') {
      console.log('Service is healthy');
    } else {
      console.warn('Service is degraded:', data);
    }
  });
```

**Python (requests)**
```python
import requests

response = requests.get('http://localhost:8081/health')
data = response.json()

if data['status'] == 'healthy':
    print('Service is healthy')
    print(f"Active connections: {data['websocket']['active_connections']}")
else:
    print('Service is degraded')
```

### 3. Metrics

Returns operational metrics for monitoring and observability.

#### Endpoint

```
GET /metrics
```

#### Response (200 OK)

```json
{
  "service": "websocket-service",
  "timestamp": "2025-01-21T10:30:00.123Z",
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

#### Response Fields

| Field                        | Type   | Description                                     |
|------------------------------|--------|-------------------------------------------------|
| service                      | string | Service name                                    |
| timestamp                    | string | ISO 8601 timestamp                              |
| uptime_seconds               | int    | Service uptime in seconds                       |
| connections.active           | int    | Total active WebSocket connections              |
| connections.total_unique_users| int   | Number of unique users connected                |
| messages.received_from_redis | int64  | Total messages received from Redis (cumulative) |
| messages.sent_to_clients     | int64  | Total messages sent to clients (cumulative)     |
| messages.failed              | int64  | Total failed message deliveries (cumulative)    |

#### Usage Examples

**cURL with jq**
```bash
curl http://localhost:8081/metrics | jq
```

**Prometheus Scraping (Custom Exporter)**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'websocket-service'
    metrics_path: '/metrics'
    static_configs:
      - targets: ['localhost:8081']
```

**Grafana Dashboard Query**
```javascript
// Active connections over time
SELECT connections.active FROM metrics

// Message throughput
SELECT derivative(messages.sent_to_clients, 1s) FROM metrics
```

## Message Formats

### WebSocket Messages (Server to Client)

All messages sent from server to client follow this structure:

```json
{
  "type": "string",
  "payload": {},
  "timestamp": "2025-01-21T10:30:00.123Z"
}
```

#### Message Structure

| Field     | Type   | Required | Description                                    |
|-----------|--------|----------|------------------------------------------------|
| type      | string | Yes      | Message type (see message types below)         |
| payload   | object | Yes      | Message content (type-specific)                |
| timestamp | string | Yes      | ISO 8601 timestamp when message was created    |

### Message Types

#### 1. Notification

General user notifications.

```json
{
  "type": "notification",
  "payload": {
    "title": "New Order",
    "message": "You have a new order #12345",
    "icon": "order",
    "link": "/orders/12345"
  },
  "timestamp": "2025-01-21T10:30:00.123Z"
}
```

**Payload Fields:**
- `title` (string): Notification title
- `message` (string): Notification message
- `icon` (string, optional): Icon identifier
- `link` (string, optional): URL to navigate to

**Use Cases:**
- New order notifications
- Message received
- Comment mentions
- Task assignments

#### 2. Alert

Important alerts requiring user attention.

```json
{
  "type": "alert",
  "payload": {
    "level": "warning",
    "message": "Your subscription expires in 3 days",
    "action": {
      "label": "Renew Now",
      "url": "/subscriptions"
    }
  },
  "timestamp": "2025-01-21T10:30:00.123Z"
}
```

**Payload Fields:**
- `level` (string): Alert level ("info", "warning", "error", "critical")
- `message` (string): Alert message
- `action` (object, optional): Action button configuration
  - `label` (string): Button label
  - `url` (string): Action URL

**Use Cases:**
- System warnings
- Security alerts
- Expiration notices
- Critical errors

#### 3. Update

Real-time data updates.

```json
{
  "type": "update",
  "payload": {
    "entity": "order",
    "entity_id": "12345",
    "status": "shipped",
    "tracking_number": "TRACK123",
    "updated_at": "2025-01-21T10:30:00Z"
  },
  "timestamp": "2025-01-21T10:30:00.123Z"
}
```

**Payload Fields:**
- `entity` (string): Entity type being updated
- `entity_id` (string): Entity identifier
- Additional fields specific to the entity

**Use Cases:**
- Order status changes
- Data synchronization
- Real-time updates
- Live dashboard data

#### 4. Custom Message Types

You can define custom message types for your specific needs:

```json
{
  "type": "order_status_changed",
  "payload": {
    "order_id": "12345",
    "old_status": "pending",
    "new_status": "shipped",
    "tracking_number": "TRACK123",
    "customer_id": "user123"
  },
  "timestamp": "2025-01-21T10:30:00.123Z"
}
```

**Custom Type Guidelines:**
- Use descriptive, snake_case names
- Include all relevant data in payload
- Document your custom types
- Keep payload size reasonable (<1KB recommended)

### Publishing Messages to Redis

Backend services publish messages to Redis channels that the WebSocket service subscribes to.

#### Channel Pattern

```
user_noti:{user_id}
```

Examples:
- `user_noti:user123` - Messages for user123
- `user_noti:admin` - Messages for admin
- `user_noti:guest_xyz` - Messages for guest_xyz

#### Message Format (Redis)

```json
{
  "type": "notification",
  "payload": {
    // Your custom payload
  }
}
```

Note: The WebSocket service adds the `timestamp` field automatically.

#### Publishing Examples

**Redis CLI**
```bash
redis-cli -a YOUR_PASSWORD
PUBLISH user_noti:user123 '{"type":"notification","payload":{"title":"Test","message":"Hello"}}'
```

**Go (go-redis)**
```go
import (
    "context"
    "encoding/json"
    "github.com/redis/go-redis/v9"
)

type RedisMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

func publishNotification(userID string, payload interface{}) error {
    client := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "YOUR_PASSWORD",
    })
    defer client.Close()
    
    msg := RedisMessage{
        Type:    "notification",
        Payload: payload,
    }
    
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    
    channel := "user_noti:" + userID
    return client.Publish(context.Background(), channel, data).Err()
}
```

**Python (redis-py)**
```python
import redis
import json

def publish_notification(user_id, payload):
    r = redis.Redis(host='localhost', port=6379, password='YOUR_PASSWORD')
    
    message = {
        'type': 'notification',
        'payload': payload
    }
    
    channel = f'user_noti:{user_id}'
    r.publish(channel, json.dumps(message))
```

**Node.js (ioredis)**
```javascript
const Redis = require('ioredis');

async function publishNotification(userId, payload) {
  const redis = new Redis({
    host: 'localhost',
    port: 6379,
    password: 'YOUR_PASSWORD'
  });
  
  const message = {
    type: 'notification',
    payload: payload
  };
  
  const channel = `user_noti:${userId}`;
  await redis.publish(channel, JSON.stringify(message));
  await redis.quit();
}
```

## Error Codes

### HTTP Error Codes

| Code | Status               | Description                                      |
|------|----------------------|--------------------------------------------------|
| 401  | Unauthorized         | Missing or invalid JWT token                     |
| 503  | Service Unavailable  | Service degraded (e.g., Redis disconnected)      |
| 500  | Internal Server Error| Unexpected server error                          |

### WebSocket Close Codes

| Code | Reason                | Description                                      |
|------|-----------------------|--------------------------------------------------|
| 1000 | Normal Closure        | Client or server initiated normal close          |
| 1001 | Going Away            | Server shutting down                             |
| 1006 | Abnormal Closure      | Connection lost (network failure)                |
| 1008 | Policy Violation      | Connection limit reached                         |
| 1011 | Internal Error        | Server internal error                            |

## Rate Limits

Currently, no rate limits are enforced on the WebSocket connection itself. However:

- **Connection Limit**: Maximum 10,000 concurrent connections per instance (configurable)
- **Message Size Limit**: Maximum 512 bytes per message (configurable)
- **Buffer Limit**: 256 messages per connection send buffer

## Best Practices

### Client Implementation

**1. Implement Reconnection Logic**
```javascript
function connectWebSocket() {
  const ws = new WebSocket(`ws://localhost:8081/ws?token=${getToken()}`);
  
  ws.onclose = (event) => {
    console.log('Connection closed, reconnecting in 5s...');
    setTimeout(connectWebSocket, 5000);
  };
  
  return ws;
}
```

**2. Handle Token Expiration**
```javascript
// Refresh token before expiration
const tokenExpiresIn = getTokenExpiration() - Date.now();
if (tokenExpiresIn < 5 * 60 * 1000) { // 5 minutes
  refreshTokenAndReconnect();
}
```

**3. Implement Exponential Backoff**
```javascript
let reconnectDelay = 1000;

function reconnect() {
  setTimeout(() => {
    connectWebSocket();
    reconnectDelay = Math.min(reconnectDelay * 2, 30000); // Max 30s
  }, reconnectDelay);
}
```

**4. Handle Multiple Tabs**
```javascript
// Use BroadcastChannel to sync across tabs
const channel = new BroadcastChannel('notifications');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  // Process locally
  handleMessage(message);
  
  // Broadcast to other tabs
  channel.postMessage(message);
};

// Listen for messages from other tabs
channel.onmessage = (event) => {
  handleMessage(event.data);
};
```

### Server Integration

**1. Validate User ID**
```go
func publishNotification(userID string, payload interface{}) error {
    // Always validate user exists
    if !userExists(userID) {
        return errors.New("invalid user ID")
    }
    
    // Publish message
    return publishToRedis(userID, payload)
}
```

**2. Use Connection Pooling**
```go
// Global Redis client with connection pooling
var redisClient *redis.Client

func init() {
    redisClient = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        PoolSize: 100,
    })
}
```

**3. Publish Asynchronously**
```go
// Don't block main operation on notification
go func() {
    if err := publishNotification(userID, payload); err != nil {
        log.Printf("Failed to publish notification: %v", err)
    }
}()
```

**4. Handle Offline Users**
```
// WebSocket service automatically skips offline users
// If you need persistent notifications, implement your own queue
if userIsOnline(userID) {
    publishNotification(userID, payload)
} else {
    saveToNotificationQueue(userID, payload)
}
```

## Testing

### Testing WebSocket Connection

**Using Postman:**
1. Create new WebSocket Request
2. URL: `ws://localhost:8081/ws?token=YOUR_TOKEN`
3. Connect
4. Publish message via Redis CLI to see it appear

**Using websocat (CLI)**
```bash
websocat "ws://localhost:8081/ws?token=YOUR_TOKEN"
```

**Using Browser Console**
```javascript
const ws = new WebSocket('ws://localhost:8081/ws?token=YOUR_TOKEN');
ws.onmessage = (e) => console.log(JSON.parse(e.data));
```

### Testing Message Delivery

**Terminal 1: Start Service**
```bash
make run
```

**Terminal 2: Connect Client**
```bash
go run tests/client_example.go YOUR_TOKEN
```

**Terminal 3: Publish Message**
```bash
redis-cli -a YOUR_PASSWORD
PUBLISH user_noti:user123 '{"type":"notification","payload":{"title":"Test"}}'
```

## Troubleshooting

### Connection Issues

**Problem: Cannot connect to WebSocket**

Check:
1. Is service running? `curl http://localhost:8081/health`
2. Is token valid? Decode at https://jwt.io/
3. Is token expired? Check `exp` claim
4. Network connectivity?

**Problem: Connection closes immediately**

Likely causes:
- Invalid JWT token
- Token expired
- Connection limit reached
- Server shutdown

### Message Delivery Issues

**Problem: Messages not received**

Check:
1. Is client still connected? Check network tab
2. Is user ID in token matching channel?
3. Check server logs for errors
4. Verify Redis message format

**Problem: Duplicate messages**

Causes:
- Multiple WebSocket connections (multiple tabs) - this is expected
- Client not handling duplicates properly

### Performance Issues

**Problem: High latency**

Check:
1. Redis connection latency: `/health` endpoint
2. Network conditions
3. Connection count: `/metrics` endpoint
4. Server resource usage

## Support

For additional help:
- Check server logs for detailed error messages
- Review `/health` endpoint for system status
- Check `/metrics` for operational metrics
- Review documentation in `document/` folder

