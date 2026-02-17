# API & Data Contracts

This document defines the strict **JSON schemas** for data flowing into and out of the `notification-srv`. It serves as the integration contract between:

1. **Backend Services** (Crawler, Analyzer, Knowledge) → `notification-srv` (via Redis)
2. **`notification-srv`** → **Frontend Client** (via WebSocket)

---

## 1. Connection Contract (WebSocket)

**Endpoint:** `GET /ws`

### Authentication

Client **MUST** provide a valid JWT via one of:

- **Cookie:** `smap_auth_token` (HttpOnly, Secure) — *Recommended*
- **Query Param:** `?token=eyJhbG...` — *Debugging only*

### Query Parameters

- `project_id` (optional): Filter messages to a specific project.

---

## 2. Input Contract (Redis Pub/Sub)

Backend services publish messages to Redis channels. `notification-srv` subscribes and routes them.

**Channel Patterns:**

- Project Scope: `project:{project_id}:user:{user_id}`
- Campaign Scope: `campaign:{campaign_id}:user:{user_id}`
- System Alert: `alert:crisis:user:{user_id}`
- System Scope: `system:{subtype}`

### 2.1 Data Onboarding Event

**Channel:** `project:{id}:user:{uid}`
**Type:** `DATA_ONBOARDING`

```json
{
  "project_id": "proj_123",
  "source_id": "src_456",
  "source_name": "My TikTok Page",
  "source_type": "tiktok",
  "status": "COMPLETED",  // PENDING, COMPLETED, FAILED
  "progress": 100,
  "record_count": 1500,
  "error_count": 0,
  "message": "Successfully imported 1500 records"
}
```

### 2.2 Analytics Pipeline Event

**Channel:** `project:{id}:user:{uid}`
**Type:** `ANALYTICS_PIPELINE`

```json
{
  "project_id": "proj_123",
  "source_id": "src_456",
  "total_records": 1000,
  "processed_count": 350,
  "success_count": 345,
  "failed_count": 5,
  "progress": 35,
  "current_phase": "ANALYZING", // CRAWLING, CLEANING, ANALYZING, INDEXING
  "estimated_time_ms": 120000
}
```

### 2.3 Crisis Alert

**Channel:** `alert:crisis:user:{uid}`
**Type:** `CRISIS_ALERT`

```json
{
  "project_id": "proj_123",
  "project_name": "VinFast VF8 Monitor",
  "severity": "CRITICAL",       // CRITICAL, WARNING, INFO
  "alert_type": "sentiment_spike",
  "metric": "Negative Sentiment",
  "current_value": 0.85,
  "threshold": 0.70,
  "affected_aspects": ["BATTERY", "PRICE"],
  "sample_mentions": [
    "Battery drains too fast",
    "Price is too high for this range"
  ],
  "time_window": "last_1h",
  "action_required": "Immediate review required"
}
```

### 2.4 Campaign Event

**Channel:** `campaign:{id}:user:{uid}`
**Type:** `CAMPAIGN_EVENT`

```json
{
  "campaign_id": "camp_abc",
  "campaign_name": "Summer Sale 2026",
  "event_type": "FINISHED",
  "resource_id": "res_xyz",    // Optional (e.g., report ID)
  "resource_name": "Campaign Report Q2",
  "resource_url": "https://smap.dev/reports/123",
  "message": "Campaign analysis completed successfully."
}
```

---

## 3. Output Contract (WebSocket Frames)

All messages sent to the browser follow this uniform structure.

### Envelope Structure

```json
{
  "type": "MESSAGE_TYPE_ENUM",
  "timestamp": "2026-02-17T14:00:00Z",
  "payload": { ... } // Varies by type
}
```

### 3.1 Data Onboarding Out

`"type": "DATA_ONBOARDING"`

```json
{
  "type": "DATA_ONBOARDING",
  "timestamp": "...",
  "payload": {
    "project_id": "proj_123",
    "source_id": "src_456",
    "source_name": "My TikTok Page",
    "status": "COMPLETED",
    "progress": 100,
    "record_count": 1500,
    // ... same as input
  }
}
```

### 3.2 Analytics Pipeline Out

`"type": "ANALYTICS_PIPELINE"`

```json
{
  "type": "ANALYTICS_PIPELINE",
  "timestamp": "...",
  "payload": {
    "project_id": "proj_123",
    "progress": 35,
    "current_phase": "ANALYZING",
    "estimated_time_ms": 120000,
    // ... same as input
  }
}
```

### 3.3 Crisis Alert Out

`"type": "CRISIS_ALERT"`

```json
{
  "type": "CRISIS_ALERT",
  "timestamp": "...",
  "payload": {
    "project_id": "proj_123",
    "severity": "CRITICAL",
    "metric": "Negative Sentiment",
    "current_value": 0.85,
    "threshold": 0.70,
    "affected_aspects": ["BATTERY", "PRICE"],
    // ... same as input
  }
}
```

---

## 4. Output Contract (Discord Alerts)

### 4.1 Crisis Alert (Rich Embed)

- **Color**: Red (Critical), Orange (Warning)
- **Title**: 🚨 Crisis Alert: {ProjectName}
- **Description**: Unusual activity detected in project **{ProjectName}**.
- **Fields**:
  - **Severity**: CRITICAL
  - **Metric**: Negative Sentiment
  - **Value**: **0.85** / 0.70
  - **Affected Aspects**: BATTERY, PRICE
  - **Sample Mentions**: "> Battery drains...", "> Price is..."
  - **Action**: Immediate review required

### 4.2 Data Onboarding

- **Color**: Green (Completed), Red (Failed)
- **Title**: Data Onboarding: {Status}
- **Fields**:
  - **Source**: {Name} ({Type})
  - **Records**: {Count}
  - **Errors**: {Count}

---

**Last Updated**: 17/02/2026
