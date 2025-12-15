# websocket-messages Specification

## Purpose
TBD - created by archiving change align-websocket-event-driven. Update Purpose after archive.
## Requirements
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

The WebSocket Service MUST validate progress payload structure before forwarding, supporting both legacy and phase-based formats.

**Legacy Payload Fields (unchanged):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `status` | string | Yes | Project status |
| `total` | int | Yes | Total items to process |
| `done` | int | Yes | Completed items |
| `errors` | int | Yes | Failed items |
| `progress_percent` | float | Yes | Calculated progress (0.0-100.0) |

#### Scenario: Detect and route phase-based format

**Given** a message is received from Redis Pub/Sub  
**And** the message has `type` field with value `project_progress` or `project_completed`  
**And** the payload contains `crawl` or `analyze` fields  
**Then** the message is identified as phase-based format  
**And** is processed using phase-based validation rules

#### Scenario: Backward compatible legacy format handling

**Given** a message is received from Redis Pub/Sub  
**And** the message does NOT have a `type` field with phase-based values  
**And** the payload contains flat `total`, `done`, `errors` fields  
**Then** the message is identified as legacy format  
**And** is processed using existing legacy validation rules  
**And** clients receive the legacy format unchanged

---

### Requirement: Message Type Enum Definition

The WebSocket message type enum MUST include the following values:

| Type                | Description                                                    |
| ------------------- | -------------------------------------------------------------- |
| `notification`      | Generic notification                                           |
| `alert`             | Alert message                                                  |
| `update`            | Update message                                                 |
| `ping`              | Client heartbeat                                               |
| `pong`              | Server heartbeat response                                      |
| `project_progress`  | Project progress update (supports both legacy and phase-based) |
| `project_completed` | Project completion (supports both legacy and phase-based)      |

#### Scenario: Status enum includes new phase-based values

**Given** the WebSocket service status enums are defined  
**When** validating project status values  
**Then** `INITIALIZING` is a valid status  
**And** `PROCESSING` is a valid status  
**And** legacy statuses `CRAWLING`, `COMPLETED`, `PAUSED` remain valid for backward compatibility

### Requirement: Phase-Based Progress Payload Support

The WebSocket Service MUST support phase-based progress payloads that include separate progress tracking for crawl and analyze phases.

**New Payload Structure:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | string | Yes | Project identifier |
| `status` | string | Yes | Project status (INITIALIZING, PROCESSING, DONE, FAILED) |
| `crawl` | PhaseProgress | No | Progress for crawl phase |
| `analyze` | PhaseProgress | No | Progress for analyze phase |
| `overall_progress_percent` | float | Yes | Combined progress (0.0-100.0) |

**PhaseProgress Structure:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `total` | int | Yes | Total items in this phase |
| `done` | int | Yes | Completed items in this phase |
| `errors` | int | Yes | Failed items in this phase |
| `progress_percent` | float | Yes | Phase progress (0.0-100.0) |

#### Scenario: Client receives phase-based progress update

**Given** a client is connected via WebSocket  
**And** the user has an executing project in PROCESSING status  
**When** Project Service publishes to `project:{projectID}:{userID}` with type `project_progress`  
**Then** the client receives a WebSocket message with:

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

#### Scenario: Client receives phase-based completion notification

**Given** a client is connected via WebSocket  
**And** the user has an executing project  
**When** Project Service publishes to `project:{projectID}:{userID}` with type `project_completed`  
**Then** the client receives a WebSocket message with:

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

#### Scenario: Phase-based message with missing optional phase

**Given** a message arrives with `crawl` phase but no `analyze` phase (project still initializing)  
**When** the message is processed  
**Then** the client receives the message with only `crawl` data  
**And** `analyze` field is omitted from the payload

---

