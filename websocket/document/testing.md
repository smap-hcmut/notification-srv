# Testing Guide - WebSocket Service

## Complete Testing Instructions

This guide walks you through testing the WebSocket service step by step.

---

## Prerequisites

### 1. Start Redis

The service requires Redis to be running.

**Option A: Using Redis CLI**
```bash
redis-server --requirepass 21042004
```

**Option B: Using Docker**
```bash
docker run -d --name redis-test \
  -p 6379:6379 \
  redis:7-alpine \
  redis-server --requirepass 21042004
```

**Verify Redis is running:**
```bash
redis-cli -a 21042004 ping
# Should respond: PONG
```

### 2. Generate JWT Token for Testing

You need a JWT token with `sub` claim containing a user ID.

**Quick Test Token Generator (Go):**

Create `generate_token.go`:
```go
package main

import (
    "fmt"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

func main() {
    // Secret key (must match JWT_SECRET_KEY in .env)
    secretKey := "my-super-secret-jwt-key-for-testing-change-in-production"
    
    // Create claims
    claims := jwt.MapClaims{
        "sub":   "user123",           // User ID
        "email": "test@example.com",  // Optional
        "exp":   time.Now().Add(24 * time.Hour).Unix(), // Expires in 24 hours
        "iat":   time.Now().Unix(),
    }
    
    // Create token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, _ := token.SignedString([]byte(secretKey))
    
    fmt.Println("Generated JWT Token:")
    fmt.Println(tokenString)
}
```

**Generate token:**
```bash
go run generate_token.go
```

**Or use online tool:** https://jwt.io/
- Algorithm: HS256
- Payload:
```json
{
  "sub": "user123",
  "email": "test@example.com",
  "exp": 1800000000,
  "iat": 1700000000
}
```
- Secret: `my-super-secret-jwt-key-for-testing-change-in-production`

## Test Scenario 1: Basic Connection Test

### Step 1: Start the WebSocket Service

**Terminal 1:**
```bash
make run
# Or: go run ./cmd/server
```

**Expected Output:**
```
{"level":"info","ts":...,"msg":"Starting WebSocket Service..."}
{"level":"info","ts":...,"msg":"Redis connected successfully to localhost:6379"}
{"level":"info","ts":...,"msg":"WebSocket Hub started"}
{"level":"info","ts":...,"msg":"Redis Pub/Sub subscriber started"}
{"level":"info","ts":...,"msg":"WebSocket server listening on 0.0.0.0:8081"}
```

### Step 2: Test Health Endpoint

**Terminal 2:**
```bash
curl http://localhost:8081/health | jq
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-21T...",
  "redis": {
    "status": "connected",
    "ping_ms": 1.23
  },
  "websocket": {
    "active_connections": 0,
    "total_unique_users": 0
  },
  "uptime_seconds": 5
}
```

### Step 3: Test Metrics Endpoint

```bash
curl http://localhost:8081/metrics | jq
```

**Expected Response:**
```json
{
  "service": "websocket-service",
  "timestamp": "2025-01-21T...",
  "uptime_seconds": 10,
  "connections": {
    "active": 0,
    "total_unique_users": 0
  },
  "messages": {
    "received_from_redis": 0,
    "sent_to_clients": 0,
    "failed": 0
  }
}
```

## Test Scenario 2: WebSocket Connection Test

### Step 1: Connect with Example Client

**Terminal 2:**
```bash
# Replace YOUR_JWT_TOKEN with the token you generated
export JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

go run tests/client_example.go $JWT_TOKEN
```

**Expected Output:**
```
Connecting to ws://localhost:8081/ws?token=eyJ...
Connected successfully!
```

**In Terminal 1 (Server logs):**
```
{"level":"info","msg":"User connected: user123 (total connections: 1, user connections: 1)"}
```

### Step 2: Verify Connection in Metrics

**Terminal 3:**
```bash
curl http://localhost:8081/metrics | jq '.connections'
```

**Expected Response:**
```json
{
  "active": 1,
  "total_unique_users": 1
}
```

## Test Scenario 3: Send Messages via Redis

### Step 1: Keep WebSocket Client Connected

Make sure the client from Scenario 2 is still running.

### Step 2: Publish Message via Redis CLI

**Terminal 3:**
```bash
redis-cli -a 21042004

# Publish a notification to user123
PUBLISH user_noti:user123 '{"type":"notification","payload":{"title":"Hello","body":"This is a test message"}}'
```

**Expected Output in Client (Terminal 2):**
```
Received: {"type":"notification","payload":{"title":"Hello","body":"This is a test message"},"timestamp":"2025-01-21T..."}
```

**Expected Output in Server (Terminal 1):**
```
{"level":"debug","msg":"Routed message to user user123 (type: notification)"}
```

### Step 3: Test Different Message Types

```bash
# Alert
PUBLISH user_noti:user123 '{"type":"alert","payload":{"level":"warning","message":"System alert"}}'

# Update
PUBLISH user_noti:user123 '{"type":"update","payload":{"entity":"order","status":"shipped"}}'

# Custom type
PUBLISH user_noti:user123 '{"type":"order_created","payload":{"order_id":"12345","total":99.99}}'
```

All messages should appear in the client.

## Test Scenario 4: Multiple Connections (Multiple Tabs)

### Step 1: Open Multiple Clients

**Terminal 2:**
```bash
go run tests/client_example.go $JWT_TOKEN
```

**Terminal 3:**
```bash
go run tests/client_example.go $JWT_TOKEN
```

**Terminal 4:**
```bash
go run tests/client_example.go $JWT_TOKEN
```

**Server should show:**
```
{"level":"info","msg":"User connected: user123 (total connections: 1, user connections: 1)"}
{"level":"info","msg":"User connected: user123 (total connections: 2, user connections: 2)"}
{"level":"info","msg":"User connected: user123 (total connections: 3, user connections: 3)"}
```

### Step 2: Send One Message

```bash
redis-cli -a 21042004
PUBLISH user_noti:user123 '{"type":"notification","payload":{"title":"Broadcast","body":"Goes to all tabs"}}'
```

**Expected:** All 3 clients should receive the message simultaneously.

### Step 3: Close One Client

Press `Ctrl+C` in one of the client terminals.

**Server should show:**
```
{"level":"info","msg":"User connection closed: user123 (remaining connections: 2)"}
```

## Test Scenario 5: Authentication Tests

### Test 1: Missing Token

```bash
# Using curl
curl -i -N -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: $(echo $RANDOM | base64)" \
  http://localhost:8081/ws
```

**Expected:** HTTP 401 Unauthorized with error message.

### Test 2: Invalid Token

```bash
curl "http://localhost:8081/ws?token=invalid-token-here"
```

**Expected:** HTTP 401 with "invalid or expired token" error.

### Test 3: Expired Token

Generate a token with past expiration time and try to connect.

**Expected:** HTTP 401 with token expired error.

## Test Scenario 6: Postman Testing

### Setup

1. Open Postman
2. Create new **WebSocket Request** (not HTTP request)
3. Enter URL: `ws://localhost:8081/ws?token=YOUR_JWT_TOKEN`
4. Click **Connect**

### Expected Result

- Connection Status: **Connected** (green)
- You can see connection logs in server

### Send Test Message

In another terminal:
```bash
redis-cli -a 21042004
PUBLISH user_noti:user123 '{"type":"notification","payload":{"title":"From Redis","body":"Received in Postman"}}'
```

**Expected:** Message appears in Postman's message list.

## Test Scenario 7: Offline User Test

### Step 1: Close All Clients

Make sure no clients are connected for user123.

### Step 2: Publish Message

```bash
redis-cli -a 21042004
PUBLISH user_noti:user123 '{"type":"notification","payload":{"title":"Nobody here","body":"This message is lost"}}'
```

**Expected Server Behavior:** Message is silently skipped (no error).

**Server logs:** No routing message (debug level might show it was received but not delivered).

### Step 3: Check Metrics

```bash
curl http://localhost:8081/metrics | jq '.messages'
```

**Expected:**
```json
{
  "received_from_redis": 1,
  "sent_to_clients": 0,
  "failed": 0
}
```

## Test Scenario 8: Stress Test (Optional)

### Concurrent Connections Test

**Create stress test script** `stress_test.sh`:
```bash
#!/bin/bash
TOKEN="YOUR_JWT_TOKEN"
for i in {1..100}; do
  go run tests/client_example.go $TOKEN &
done
wait
```

```bash
chmod +x stress_test.sh
./stress_test.sh
```

**Check metrics:**
```bash
curl http://localhost:8081/metrics | jq '.connections'
# Should show 100 active connections
```

## Test Scenario 9: Graceful Shutdown

### Step 1: Connect Client

```bash
go run tests/client_example.go $JWT_TOKEN
```

### Step 2: Stop Server

In Terminal 1 (server), press `Ctrl+C`

**Expected Server Output:**
```
{"level":"info","msg":"Shutting down gracefully..."}
{"level":"info","msg":"Hub shutting down..."}
{"level":"info","msg":"Closed all connections for user: user123"}
{"level":"info","msg":"Shutting down HTTP server..."}
{"level":"info","msg":"Server shutdown complete"}
```

**Expected Client Output:**
```
Read error: websocket: close 1006 (abnormal closure): unexpected EOF
```

## Troubleshooting

### Problem: Can't connect to WebSocket

**Check:**
```bash
# 1. Is service running?
curl http://localhost:8081/health

# 2. Is Redis running?
redis-cli -a 21042004 ping

# 3. Is JWT token valid?
# Decode at https://jwt.io/ and check expiration
```

### Problem: Messages not received

**Check:**
1. Is user ID in token (`sub` claim) matching the channel?
2. Is client still connected?
3. Check server logs for errors
4. Try publishing directly: `redis-cli -a 21042004` then `PUBLISH user_noti:user123 '{"type":"test","payload":{}}'`

### Problem: Redis connection failed

**Fix:**
```bash
# Check Redis is running
ps aux | grep redis

# Check password
redis-cli -a 21042004 ping

# Check .env file has correct REDIS_PASSWORD
cat .env | grep REDIS_PASSWORD
```

## Testing Checklist

- [ ] Service starts without errors
- [ ] Health endpoint returns 200 OK
- [ ] Metrics endpoint shows correct data
- [ ] Client can connect with valid JWT
- [ ] Client rejected with invalid JWT
- [ ] Messages delivered from Redis to client
- [ ] Multiple clients receive same message
- [ ] Connection cleanup on disconnect
- [ ] Graceful shutdown works
- [ ] Offline users don't cause errors

## Next Steps

After successful testing:

1. **Deploy to production**
2. **Integrate other services** (see `document/integration.md`)
3. **Monitor metrics** in production
4. **Set up alerts** for health check failures
5. **Scale horizontally** if needed

**Happy Testing!**
