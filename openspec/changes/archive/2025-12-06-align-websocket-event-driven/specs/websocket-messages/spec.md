# WebSocket Messages Capability Spec

**Capability**: `websocket-messages`  
**Owner**: WebSocket Service Team

---

## Overview

Defines the message types and payload schemas for WebSocket communication between WebSocket Service and connected clients.

---

## ADDED Requirements

### Requirement: Project Progress Message Types

The WebSocket Service MUST support project execution progress notifications from Project Service.

**Message Types**:

- `project_progress` - Progress update during project execution
- `project_completed` - Final notification when project finishes

#### Scenario: Client receives project progress update

**Given** a client is connected via WebSocket  
**And** the user has an executing project  
**When** Project Service publishes to `user_noti:{userID}` with type `project_progress`  
**Then** the client receives a WebSocket message with:

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

#### Scenario: Client receives project completion notification

**Given** a client is connected via WebSocket  
**And** the user has an executing project  
**When** Project Service publishes to `user_noti:{userID}` with type `project_completed`  
**Then** the client receives a WebSocket message with:

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
  "timestamp": "2025-12-07T12:00:00Z"
}
```

---

### Requirement: Progress Payload Schema Validation

The WebSocket Service MUST validate progress payload structure before forwarding.

**Payload Fields**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `status` | string | Yes | Project status (INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED) |
| `total` | int | Yes | Total items to process |
| `done` | int | Yes | Completed items |
| `errors` | int | Yes | Failed items |
| `progress_percent` | float | Yes | Calculated progress (0.0-100.0) |

#### Scenario: Valid progress payload is forwarded

**Given** a message is received from Redis Pub/Sub  
**And** the message type is `project_progress`  
**And** the payload contains all required fields  
**Then** the message is forwarded to the connected client

#### Scenario: Malformed payload is logged but not forwarded

**Given** a message is received from Redis Pub/Sub  
**And** the payload cannot be parsed as JSON  
**Then** an error is logged  
**And** the message is NOT forwarded to clients

---

### Requirement: Message Type Enum Definition

The WebSocket message type enum MUST include the following values:

| Type                | Description                          |
| ------------------- | ------------------------------------ |
| `notification`      | Generic notification                 |
| `alert`             | Alert message                        |
| `update`            | Update message                       |
| `ping`              | Client heartbeat                     |
| `pong`              | Server heartbeat response            |
| `project_progress`  | Project progress update              |
| `project_completed` | Project completion                   |

#### Scenario: Message type enum includes all required values

**Given** the WebSocket service message types are defined  
**When** referencing `MessageTypeProjectProgress`  
**Then** the value equals `"project_progress"`  
**And** referencing `MessageTypeProjectCompleted` equals `"project_completed"`

---

## Related Capabilities

- [subscriber-status](../subscriber-status/spec.md) - Health check for Redis subscriber
- [event-driven-architecture](../../../document/event-drivent.md) - System-wide event specification
