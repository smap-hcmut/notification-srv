# Message Transformation Layer

## Purpose

Defines requirements for the transform layer that converts Redis Pub/Sub messages from input format to standardized output format before delivery to WebSocket clients. Covers validation, normalization, error handling, and performance requirements.
## Requirements
### Requirement: Mandatory Message Transformation MUST Be Implemented
All messages received from Redis MUST be transformed from input format to standardized output format before delivery to clients.

#### Scenario: Project Message Transformation
**Given** a Redis message arrives on `project:proj_123:user_456` with payload:
```json
{
  "status": "PROCESSING",
  "progress": {
    "current": 800,
    "total": 1000,
    "percentage": 80.0,
    "eta": 8.5,
    "errors": []
  }
}
```
**When** the transform layer processes the message  
**Then** it must output a standardized ProjectNotificationMessage structure  
**And** must preserve all field values exactly  
**And** must handle omitempty fields correctly (omit progress if null/empty)

#### Scenario: Job Message Transformation with Batch Data
**Given** a Redis message arrives on `job:job_789:user_456` with payload containing batch and progress data  
**When** the transform layer processes the message  
**Then** it must transform to JobNotificationMessage structure  
**And** must preserve platform enum values exactly  
**And** must transform content_list array to ContentItem structures  
**And** must handle optional media fields with omitempty behavior

#### Scenario: Transform Input Validation
**Given** a Redis message arrives with invalid JSON structure  
**When** the transform layer attempts to process it  
**Then** it must log a validation error with the channel name and error details  
**And** must not crash the service or affect other message processing  
**And** must increment transform error metrics

#### Scenario: Missing Required Fields Handling
**Given** a Redis message arrives missing the required "status" field  
**When** the transform layer validates the input  
**Then** it must reject the message and log a validation error  
**And** must specify which required field is missing  
**And** must not attempt to transform or deliver the message

### Requirement: Transform Error Handling SHALL Be Implemented
The transform layer MUST handle errors gracefully without affecting service stability.

#### Scenario: JSON Parse Error Recovery
**Given** a Redis message with malformed JSON arrives  
**When** the transform layer encounters a parse error  
**Then** it must log the error with channel and payload details  
**And** must continue processing other messages normally  
**And** must not propagate the error to the hub or clients

#### Scenario: Type Conversion Error Handling
**Given** a Redis message with incorrect data types (string where number expected)  
**When** the transform layer processes the message  
**Then** it must log a type conversion error  
**And** must attempt graceful fallback where possible  
**And** must skip the message if conversion is not possible

#### Scenario: Transform Performance Monitoring
**Given** messages are being processed by the transform layer  
**When** transformation occurs  
**Then** it must track transformation latency metrics  
**And** must count successful vs failed transformations  
**And** must provide metrics by topic type (project, job)

### Requirement: Field Normalization and Validation MUST Be Enforced
The transform layer MUST normalize and validate field values during transformation.

#### Scenario: Status Enum Validation
**Given** a project message with status "processing" (lowercase)  
**When** transforming the message  
**Then** it must normalize to "PROCESSING" (uppercase)  
**And** must validate against allowed enum values  
**And** must reject messages with invalid status values

#### Scenario: Progress Field Validation  
**Given** a message with progress.percentage > 100 or < 0  
**When** validating the progress fields  
**Then** it must clamp percentage to valid range 0-100  
**And** must log a warning about invalid percentage value  
**And** must continue processing the message with corrected value

#### Scenario: Content List Deduplication
**Given** a job message with duplicate content IDs in content_list  
**When** transforming the batch data  
**Then** it must identify and log duplicate content IDs  
**And** must preserve only the first occurrence of each unique ID  
**And** must maintain content order for remaining items

### Requirement: Transform Layer Output Standardization SHALL Be Enforced
All transformed messages MUST conform to exact output structure specifications.

#### Scenario: Project Output Structure Compliance
**Given** any valid project input message  
**When** transformed by the transform layer  
**Then** the output must conform exactly to ProjectNotificationMessage type definition  
**And** must include all required fields (status)  
**And** must use omitempty for optional fields (progress)  
**And** must use correct enum types (ProjectStatus)

#### Scenario: Job Output Structure Compliance
**Given** any valid job input message  
**When** transformed by the transform layer  
**Then** the output must conform exactly to JobNotificationMessage type definition  
**And** must preserve platform enum values (TIKTOK, YOUTUBE, INSTAGRAM)  
**And** must transform all nested structures (AuthorInfo, EngagementMetrics, MediaInfo)  
**And** must handle optional fields correctly throughout the structure

#### Scenario: Consistent Field Naming
**Given** any transformed message  
**When** serialized to JSON for client delivery  
**Then** field names must match exactly between input and output specifications  
**And** must use snake_case for JSON field names where specified  
**And** must use camelCase for JSON field names where specified  
**And** must maintain consistent naming conventions across all message types

### Requirement: Transform Performance Requirements MUST Be Met
The transform layer MUST meet performance requirements for high-throughput scenarios.

#### Scenario: Transform Latency Limits
**Given** a message requiring transformation  
**When** processed by the transform layer  
**Then** transformation must complete within 3ms for 95% of messages  
**And** must complete within 10ms for 99% of messages  
**And** must never block other message processing

#### Scenario: Memory Usage Control
**Given** high-volume message processing  
**When** the transform layer operates continuously  
**Then** it must not accumulate memory over time (no memory leaks)  
**And** must reuse transformation structures where possible  
**And** must release references to processed messages immediately

#### Scenario: Concurrent Transform Processing
**Given** multiple messages arriving simultaneously  
**When** the transform layer processes them  
**Then** it must support concurrent transformation without race conditions  
**And** must maintain message order within each topic channel  
**And** must not corrupt transformation results due to concurrency

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

