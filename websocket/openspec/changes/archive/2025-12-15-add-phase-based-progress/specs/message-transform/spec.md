# message-transform Spec Delta

## ADDED Requirements

### Requirement: Phase-Based Message Detection and Routing MUST Be Implemented

The transform layer MUST detect and correctly route phase-based messages vs legacy messages.

#### Scenario: Detect phase-based message format

**Given** a Redis message arrives with JSON payload  
**And** the payload contains a `type` field with value `project_progress` or `project_completed`  
**When** the transform layer inspects the message  
**Then** it must identify this as a phase-based format message  
**And** route to the phase-based transformer

#### Scenario: Detect legacy message format

**Given** a Redis message arrives with JSON payload  
**And** the payload does NOT contain a `type` field, or `type` is not phase-based values  
**When** the transform layer inspects the message  
**Then** it must identify this as a legacy format message  
**And** route to the legacy transformer  
**And** process with existing transformation logic

---

### Requirement: Phase-Based Message Transformation MUST Be Implemented

The transform layer MUST correctly transform phase-based input messages to standardized output format.

#### Scenario: Transform project_progress message

**Given** a Redis message arrives on `project:proj_123:user_456` with phase-based payload:

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_123",
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

**When** the transform layer processes the message  
**Then** it must output a `ProjectPhaseNotificationMessage` structure  
**And** must preserve all field values exactly  
**And** must validate all phase progress values are within valid ranges

#### Scenario: Transform project_completed message

**Given** a Redis message arrives with type `project_completed`  
**And** status is `DONE` or `FAILED`  
**When** the transform layer processes the message  
**Then** it must output a `ProjectPhaseNotificationMessage` with type `project_completed`  
**And** `overall_progress_percent` should be 100.0 for `DONE` status

#### Scenario: Phase progress validation

**Given** a phase-based message arrives with invalid progress values  
**When** the transform layer validates the input  
**Then** it must reject if `progress_percent` is outside 0-100 range  
**And** it must reject if `done` exceeds `total` (when total > 0)  
**And** it must reject if `total`, `done`, or `errors` are negative

---

### Requirement: Phase-Based Status Validation MUST Be Enforced

The transform layer MUST validate and normalize phase-based status values.

#### Scenario: New status enum validation

**Given** a phase-based message with status `PROCESSING`  
**When** validating the status field  
**Then** it must accept `INITIALIZING` as valid status  
**And** it must accept `PROCESSING` as valid status  
**And** it must accept `DONE` as valid status  
**And** it must accept `FAILED` as valid status

#### Scenario: Invalid status rejection

**Given** a phase-based message with status `INVALID_STATUS`  
**When** validating the status field  
**Then** it must reject the message  
**And** must log a validation error specifying the invalid status value

---

### Requirement: Phase-Based Transform Metrics MUST Be Collected

The transform layer MUST track separate metrics for phase-based vs legacy message processing.

#### Scenario: Track message format distribution

**Given** messages are being processed by the transform layer  
**When** transformation occurs  
**Then** it must increment counter for `phase_based` format when processing phase-based messages  
**And** it must increment counter for `legacy` format when processing legacy messages  
**And** metrics must be available for monitoring format migration progress

#### Scenario: Track phase-based transform latency

**Given** a phase-based message is being transformed  
**When** transformation completes  
**Then** it must record transformation latency with topic type `project_phase`  
**And** must track success/failure counts separately from legacy transforms
