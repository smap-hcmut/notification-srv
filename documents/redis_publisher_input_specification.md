# Redis Publisher Input Specification

**Ngày tạo**: 2025-12-14  
**Tác giả**: System Analysis  
**Trạng thái**: Ready for Implementation  
**Phiên bản**: 1.0 - Final

---

## Tổng Quan

Tài liệu này định nghĩa **expected structure** mà các Publishers phải tuân theo khi publish messages vào Redis. WebSocket service sẽ subscribe các topics này và transform messages thành format chuẩn để gửi đến clients.

### Nguyên Tắc Chính

1. **Publishers giữ nguyên format hiện tại** - Không cần thay đổi code
2. **WebSocket service transform** - Chuyển đổi từ input format sang output format
3. **Topic-based routing** - Mỗi topic có structure riêng
4. **Backward compatible** - Hỗ trợ format cũ và mới

---

## Topic Subscription Requirements

### 1. Project Topic

**Pattern**: `project:projectID:userID`  
**Example**: `project:proj_123:user_456`

**Publishers phải publish vào**: Các channels có format chính xác này  
**WebSocket subscribes**: `project:*` (pattern matching)

### 2. Job Topic

**Pattern**: `job:jobID:userID`  
**Example**: `job:job_789:user_456`

**Publishers phải publish vào**: Các channels có format chính xác này  
**WebSocket subscribes**: `job:*` (pattern matching)

---

## Expected Input Structures

### Project Topic Input Structure

#### **Channel Pattern**

```
project:{projectID}:{userID}
```

#### **Expected Message Structure**

```go
// Input structure từ Publishers - Simplified
type ProjectInputMessage struct {
    Status   string         `json:"status"`             // "PROCESSING", "COMPLETED", "FAILED", "PAUSED"
    Progress *ProgressInput `json:"progress,omitempty"` // Overall progress
}

type ProgressInput struct {
    Current    int      `json:"current"`     // Current completed items
    Total      int      `json:"total"`       // Total items to process
    Percentage float64  `json:"percentage"`  // Completion percentage (0-100)
    ETA        float64  `json:"eta"`         // Estimated time remaining in minutes
    Errors     []string `json:"errors"`      // Array of error messages
}
```

#### **Payload Structure Examples**

##### **Progress Update Message**

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

##### **Completion Message**

```json
{
  "status": "COMPLETED",
  "progress": {
    "current": 1000,
    "total": 1000,
    "percentage": 100.0,
    "eta": 0.0,
    "errors": []
  }
}
```

##### **Error Message**

```json
{
  "status": "FAILED",
  "progress": {
    "current": 450,
    "total": 1000,
    "percentage": 45.0,
    "eta": 0.0,
    "errors": ["Processing encountered 5 errors"]
  }
}
```

---

### Job Topic Input Structure

#### **Channel Pattern**

```
job:{jobID}:{userID}
```

#### **Expected Message Structure**

```go
// Input structure từ Publishers - Simplified
type JobInputMessage struct {
    Platform string     `json:"platform"`           // "TIKTOK", "YOUTUBE", "INSTAGRAM"
    Status   string     `json:"status"`             // "PROCESSING", "COMPLETED", "FAILED", "PAUSED"
    Batch    *BatchInput `json:"batch,omitempty"`   // Current batch data
    Progress *ProgressInput `json:"progress,omitempty"` // Overall job progress
}

type BatchInput struct {
    Keyword     string         `json:"keyword"`      // Search keyword
    ContentList []ContentInput `json:"content_list"` // Crawled content items
    CrawledAt   string         `json:"crawled_at"`   // ISO timestamp
}

type ProgressInput struct {
    Current    int      `json:"current"`     // Current completed batches
    Total      int      `json:"total"`       // Total batches to process
    Percentage float64  `json:"percentage"`  // Completion percentage (0-100)
    ETA        float64  `json:"eta"`         // Estimated time remaining in minutes
    Errors     []string `json:"errors"`      // Array of aggregated error messages
}
```

#### **Content Input Structure**

```go
// Simplified content structure for Publishers
type ContentInput struct {
    ID          string        `json:"id"`          // Content unique ID
    Text        string        `json:"text"`        // Content text/caption
    Author      AuthorInput   `json:"author"`      // Author information
    Metrics     MetricsInput  `json:"metrics"`     // Engagement statistics
    Media       *MediaInput   `json:"media,omitempty"`       // Media information
    PublishedAt string        `json:"published_at"` // ISO timestamp
    Permalink   string        `json:"permalink"`   // Direct link to content
}

type AuthorInput struct {
    ID         string `json:"id"`         // Author unique ID
    Username   string `json:"username"`   // Author username/handle
    Name       string `json:"name"`       // Author display name
    Followers  int    `json:"followers"`  // Follower count
    IsVerified bool   `json:"is_verified"` // Verification status
    AvatarURL  string `json:"avatar_url"` // Profile picture URL
}

type MetricsInput struct {
    Views    int     `json:"views"`    // View count
    Likes    int     `json:"likes"`    // Like count
    Comments int     `json:"comments"` // Comment count
    Shares   int     `json:"shares"`   // Share count
    Rate     float64 `json:"rate"`     // Engagement rate percentage
}

type MediaInput struct {
    Type      string `json:"type"`      // "video", "image", "audio"
    Duration  int    `json:"duration,omitempty"`  // Duration in seconds
    Thumbnail string `json:"thumbnail"` // Thumbnail/preview URL
    URL       string `json:"url"`       // Media file URL
}
```

#### **Job Input Examples**

##### **Batch Completion Message**

```json
{
  "platform": "TIKTOK",
  "status": "PROCESSING",
  "batch": {
    "keyword": "christmas trends",
    "content_list": [
      {
        "id": "7312345678901234567",
        "text": "Christmas decoration ideas that will blow your mind! 🎄✨ #christmas #decor #trending",
        "author": {
          "id": "user123456",
          "username": "@decorqueen",
          "name": "Sarah Johnson",
          "followers": 125000,
          "is_verified": true,
          "avatar_url": "https://example.com/avatar.jpg"
        },
        "metrics": {
          "views": 2500000,
          "likes": 180000,
          "comments": 5200,
          "shares": 12000,
          "rate": 7.88
        },
        "media": {
          "type": "video",
          "duration": 45,
          "thumbnail": "https://example.com/thumb.jpg",
          "url": "https://example.com/video.mp4"
        },
        "published_at": "2024-12-10T15:30:00Z",
        "permalink": "https://tiktok.com/@decorqueen/video/7312345678901234567"
      }
    ],
    "crawled_at": "2024-12-14T10:15:30Z"
  },
  "progress": {
    "current": 15,
    "total": 50,
    "percentage": 30.0,
    "eta": 25.5,
    "errors": [
      "Rate limit exceeded for keyword: christmas trends",
      "Failed to fetch content: network timeout",
      "Invalid response format from TikTok API"
    ]
  }
}
```

##### **Job Completion Message**

```json
{
  "platform": "TIKTOK",
  "status": "COMPLETED",
  "progress": {
    "current": 50,
    "total": 50,
    "percentage": 100.0,
    "eta": 0.0,
    "errors": []
  }
}
```

##### **Job Error Message**

```json
{
  "platform": "TIKTOK",
  "status": "FAILED",
  "progress": {
    "current": 12,
    "total": 50,
    "percentage": 24.0,
    "eta": 0.0,
    "errors": [
      "403 Forbidden: API access denied",
      "Rate limit exceeded: 1000 requests/hour",
      "Invalid API key: expired token"
    ]
  }
}
```

---

## 🔄 Transformation Mapping

### Project Input → Output Transformation

```go
// Input từ Redis
ProjectInputMessage {
    Status: "PROCESSING",
    Progress: {...}
}

// Output đến WebSocket Client
ProjectNotificationMessage {
    Status: "PROCESSING",   // Từ input.Status
    Progress: {...}         // Từ input.Progress
}
```

### Job Input → Output Transformation

```go
// Input từ Redis
JobInputMessage {
    Platform: "TIKTOK",
    Status: "PROCESSING",
    Batch: {
        Keyword: "christmas trends",
        ContentList: [...],
        CrawledAt: "2024-12-14T10:15:30Z"
    },
    Progress: {
        Current: 15,
        Total: 50,
        Percentage: 30.0,
        ETA: 25.5,
        Errors: [...]
    }
}

// Output đến WebSocket Client
JobNotificationMessage {
    Platform: "TIKTOK",             // Từ input.Platform
    Status: "PROCESSING",           // Từ input.Status
    Batch: {...},                   // Từ input.Batch (transform content structure)
    Progress: {...}                 // Từ input.Progress
}
```

---

## WebSocket Service Responsibilities

1. **Subscribe patterns**: `project:*`, `job:*`
2. **Parse channel**: Extract `projectID`/`jobID` và `userID` từ channel name
3. **Validate input**: Check required fields trong message
4. **Transform structure**: Convert input format sang output format
5. **Route message**: Gửi đến đúng WebSocket connections
6. **Handle errors**: Log transformation errors, fallback gracefully

---

**Lưu ý**: Tài liệu này định nghĩa contract giữa Publishers và WebSocket service. Publishers phải tuân theo format này để đảm bảo messages được xử lý chính xác.
