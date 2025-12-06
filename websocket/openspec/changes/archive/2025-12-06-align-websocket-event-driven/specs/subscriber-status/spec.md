# Subscriber Status Capability Spec

**Capability**: `subscriber-status`  
**Owner**: WebSocket Service Team

---

## Overview

Defines health monitoring capabilities for the Redis Pub/Sub subscriber component.

---

## ADDED Requirements

### Requirement: Subscriber Health in Health Endpoint

The `/health` endpoint MUST include subscriber status information.

**Response Schema Extension**:

```json
{
  "status": "healthy",
  "timestamp": "2025-12-07T10:00:00Z",
  "redis": { ... },
  "websocket": { ... },
  "subscriber": {
    "active": true,
    "last_message_at": "2025-12-07T09:59:55Z",
    "pattern": "user_noti:*"
  }
}
```

#### Scenario: Healthy subscriber status

**Given** the WebSocket service is running  
**And** the Redis subscriber is actively listening  
**When** a client requests `/health`  
**Then** the response includes `subscriber.active = true`  
**And** the response includes `subscriber.pattern = "user_noti:*"`

#### Scenario: Subscriber received messages

**Given** the WebSocket service is running  
**And** a message was received from Redis at time T  
**When** a client requests `/health`  
**Then** the response includes `subscriber.last_message_at` with timestamp T

#### Scenario: No messages received yet

**Given** the WebSocket service just started  
**And** no messages have been received from Redis  
**When** a client requests `/health`  
**Then** the response includes `subscriber.last_message_at` as null or omitted

---

### Requirement: Disconnect Callback Integration

The Hub MUST notify the Redis subscriber when a user fully disconnects.

#### Scenario: User closes all connections

**Given** a user has 2 WebSocket connections open  
**When** the user closes both connections  
**Then** `OnUserDisconnected(userID, hasOtherConnections=false)` is called  
**And** the subscriber removes the user from tracking

#### Scenario: User closes one of multiple connections

**Given** a user has 2 WebSocket connections open  
**When** the user closes 1 connection  
**Then** `OnUserDisconnected(userID, hasOtherConnections=true)` is called  
**And** the subscriber keeps the user in tracking

---

## Related Capabilities

- [websocket-messages](../websocket-messages/spec.md) - Message types and schemas
