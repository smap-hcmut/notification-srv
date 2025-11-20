# Integration Guide - WebSocket Notification Service

How to integrate other services with the WebSocket notification hub for real-time message delivery.

## Overview

This guide explains how to integrate your services with the WebSocket notification service to send real-time notifications to users.

**Architecture:**
```
Your Service → Redis PUBLISH → WebSocket Service → User's Browser
```

## Integration Steps

### Step 1: Connect to Redis

Your service needs access to the same Redis instance used by the WebSocket service.

**Configuration:**
- Host: `localhost` (or your Redis host)
- Port: `6379`
- Password: `21042004` (as configured)
- TLS: Enabled (if configured)

### Step 2: Publish Messages

To send a notification to a user, publish a message to their personal channel.

**Channel Pattern:** `user_noti:{user_id}`

**Message Format:**
```json
{
  "type": "notification",
  "payload": {
    // Your custom payload here
  }
}
```

## Integration Examples

### Go (golang)

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/redis/go-redis/v9"
)

type NotificationPayload struct {
    Title   string `json:"title"`
    Message string `json:"message"`
}

type RedisMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

func sendNotification(userID string, title, message string) error {
    // Create Redis client
    client := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "21042004",
        DB:       0,
    })
    defer client.Close()

    // Prepare message
    msg := RedisMessage{
        Type: "notification",
        Payload: NotificationPayload{
            Title:   title,
            Message: message,
        },
    }

    // Marshal to JSON
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    // Publish to user's channel
    channel := "user_noti:" + userID
    return client.Publish(context.Background(), channel, data).Err()
}

// Usage
func main() {
    err := sendNotification("user123", "New Order", "You have a new order #12345")
    if err != nil {
        panic(err)
    }
}
```

### Python

```python
import redis
import json

def send_notification(user_id: str, title: str, message: str):
    # Connect to Redis
    r = redis.Redis(
        host='localhost',
        port=6379,
        password='21042004',
        decode_responses=True
    )
    
    # Prepare message
    msg = {
        "type": "notification",
        "payload": {
            "title": title,
            "message": message
        }
    }
    
    # Publish to user's channel
    channel = f"user_noti:{user_id}"
    r.publish(channel, json.dumps(msg))

# Usage
send_notification("user123", "New Order", "You have a new order #12345")
```

### Node.js (JavaScript)

```javascript
const redis = require('redis');

async function sendNotification(userId, title, message) {
    // Create Redis client
    const client = redis.createClient({
        host: 'localhost',
        port: 6379,
        password: '21042004'
    });
    
    await client.connect();
    
    // Prepare message
    const msg = {
        type: 'notification',
        payload: {
            title: title,
            message: message
        }
    };
    
    // Publish to user's channel
    const channel = `user_noti:${userId}`;
    await client.publish(channel, JSON.stringify(msg));
    
    await client.disconnect();
}

// Usage
sendNotification('user123', 'New Order', 'You have a new order #12345');
```

### PHP

```php
<?php
require 'vendor/autoload.php';

use Predis\Client;

function sendNotification($userId, $title, $message) {
    // Connect to Redis
    $client = new Client([
        'scheme' => 'tcp',
        'host'   => 'localhost',
        'port'   => 6379,
        'password' => '21042004',
    ]);
    
    // Prepare message
    $msg = [
        'type' => 'notification',
        'payload' => [
            'title' => $title,
            'message' => $message
        ]
    ];
    
    // Publish to user's channel
    $channel = "user_noti:{$userId}";
    $client->publish($channel, json_encode($msg));
}

// Usage
sendNotification('user123', 'New Order', 'You have a new order #12345');
?>
```

## Message Types

### Notification

General notifications for user actions.

```json
{
  "type": "notification",
  "payload": {
    "title": "New Message",
    "body": "You have a new message from John",
    "icon": "message",
    "link": "/messages/123"
  }
}
```

### Alert

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
  }
}
```

### Update

Real-time data updates.

```json
{
  "type": "update",
  "payload": {
    "entity": "order",
    "entity_id": "12345",
    "status": "shipped",
    "timestamp": "2025-01-21T10:30:00Z"
  }
}
```

### Custom Types

You can define custom types for your specific needs:

```json
{
  "type": "order_status_changed",
  "payload": {
    "order_id": "12345",
    "old_status": "pending",
    "new_status": "shipped",
    "tracking_number": "TRACK123"
  }
}
```

## Security Considerations

### User ID Validation

Always validate user IDs before publishing:

```go
func isValidUserID(userID string) bool {
    // Validate format (UUID, numeric, etc.)
    // Check user exists in database
    // Check permissions
    return true
}

if !isValidUserID(userID) {
    return errors.New("invalid user ID")
}
```

### Message Sanitization

Sanitize message content to prevent XSS:

```go
import "html"

payload.Title = html.EscapeString(payload.Title)
payload.Message = html.EscapeString(payload.Message)
```

### Rate Limiting

Implement rate limiting to prevent abuse:

```go
// Example: max 100 notifications per user per minute
if exceedsRateLimit(userID) {
    return errors.New("rate limit exceeded")
}
```

## Best Practices

### 1. Use Connection Pooling

Reuse Redis connections instead of creating new ones:

```go
// Global Redis client
var redisClient *redis.Client

func init() {
    redisClient = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "21042004",
        PoolSize: 10,
    })
}
```

### 2. Handle Errors Gracefully

Don't fail the main operation if notification fails:

```go
// Don't do this
err := sendNotification(userID, title, msg)
if err != nil {
    return err // Main operation fails!
}

// Do this instead
go func() {
    if err := sendNotification(userID, title, msg); err != nil {
        log.Printf("Failed to send notification: %v", err)
    }
}()
```

### 3. Use Async Publishing

Publish notifications asynchronously:

```go
func createOrder(order Order) error {
    // Create order in database
    if err := db.Create(&order).Error; err != nil {
        return err
    }
    
    // Send notification asynchronously
    go sendNotification(
        order.UserID,
        "New Order",
        fmt.Sprintf("Order #%s created", order.ID),
    )
    
    return nil
}
```

### 4. Batch Notifications

For multiple users, batch operations:

```go
func notifyMultipleUsers(userIDs []string, title, message string) {
    pipeline := redisClient.Pipeline()
    
    msg := RedisMessage{Type: "notification", Payload: ...}
    data, _ := json.Marshal(msg)
    
    for _, userID := range userIDs {
        channel := "user_noti:" + userID
        pipeline.Publish(context.Background(), channel, data)
    }
    
    pipeline.Exec(context.Background())
}
```

## Testing Integration

### Test with Redis CLI

```bash
# Connect to Redis
redis-cli -a 21042004

# Publish test message
PUBLISH user_noti:user123 '{"type":"notification","payload":{"title":"Test","message":"Hello"}}'

# Monitor all channels (for debugging)
PSUBSCRIBE user_noti:*
```

### Test with Example Script

```bash
# Go
go run examples/send_notification.go user123 "Test" "Message"

# Python
python examples/send_notification.py user123 "Test" "Message"

# Node.js
node examples/send_notification.js user123 "Test" "Message"
```

## Monitoring

### Check if User is Online

```bash
# Check active connections via metrics endpoint
curl http://localhost:8081/metrics

# Response includes active users
{
  "connections": {
    "active": 1234,
    "total_unique_users": 890
  }
}
```

### Message Delivery Status

The WebSocket service follows this logic:
- ✅ User online → Message delivered immediately
- ⏭️ User offline → Message skipped (no queue)

**Note:** Messages are NOT queued. If you need persistent notifications, implement your own queue/database.

## Support

For integration issues:
1. Check Redis connectivity
2. Verify message format (must be valid JSON)
3. Confirm user is connected to WebSocket
4. Check service logs
5. Review metrics endpoint

## Quick Reference

**Channel Format:**
```
user_noti:{user_id}
```

**Message Format:**
```json
{
  "type": "string",
  "payload": {}
}
```

**Common Types:**
- `notification` - General notifications
- `alert` - Important alerts
- `update` - Data updates
- Custom types as needed

**Need help?** Check the main [README](../README.md), [OVERVIEW](OVERVIEW.md), or [API_REFERENCE](API_REFERENCE.md)
