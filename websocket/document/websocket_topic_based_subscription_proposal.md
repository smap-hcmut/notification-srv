# WebSocket Topic-Based Subscription - Đề Xuất Cuối Cùng

**Ngày tạo**: 2025-12-13  
**Tác giả**: Phân Tích Hệ Thống  
**Trạng thái**: Sẵn sàng triển khai  
**Phiên bản**: 3.0 - Cuối cùng

---

## Tóm Tắt Điều Hành

### Vấn Đề Hiện Tại

Dịch vụ WebSocket hiện tại chỉ hỗ trợ đăng ký theo `userID` với pattern `user_noti:*`. Tất cả tin nhắn của một người dùng đều được gửi đến tất cả kết nối của người dùng đó, gây ra:

- **Nhiễu tin nhắn**: Client nhận tin nhắn không liên quan
- **Lãng phí băng thông**: Gửi tin nhắn không cần thiết
- **Trải nghiệm người dùng kém**: Người dùng bị spam thông báo

### Giải Pháp Đề Xuất

Triển khai **Đăng Ký Theo Chủ Đề** với 2 chủ đề chính:

1. ~~**Chủ đề Người dùng**: `user_noti:userID` - Thông báo chung~~ _(Tạm thời bỏ qua)_
2. **Chủ đề Dự án**: `project:projectID:userID` - Thông báo cụ thể theo dự án
3. **Chủ đề Công việc**: `job:jobID:userID` - Thông báo cụ thể theo công việc

### Lợi Ích

- **Tin nhắn có mục tiêu**: Client chỉ nhận tin nhắn liên quan
- **Giảm băng thông**: Giảm 60-80% tin nhắn không cần thiết
- **Trải nghiệm người dùng tốt hơn**: Thông báo có ngữ cảnh rõ ràng
- **Có thể mở rộng**: Dễ dàng thêm chủ đề mới
- **Tương thích ngược**: Không phá vỡ các client hiện có

### ⚠️ **Bắt Buộc: Transform Layer trong WebSocket Service**

**Publishers giữ nguyên định dạng hiện tại** - Không cần thay đổi code publisher hiện có.

**WebSocket Service BẮT BUỘC phải có Transform Layer** - Chuyển đổi từ Redis input format sang standardized output format trước khi gửi đến clients.

```
Publishers ──► Redis Input Format ──► WebSocket Service ──► Transform Layer ──► Standardized Output ──► Clients
     │                                        │                    │                      │
     │ Existing message format                │ MANDATORY LAYER    │ Clean, typed structs │
     └────────────────────────────────────────┴────────────────────┴──────────────────────┴─► Better UX
```

**Transform Layer Requirements:**

- **Bắt buộc implement**: Không thể bỏ qua transform layer
- **Input validation**: Validate Redis messages trước khi transform
- **Structure mapping**: Convert từ flexible input sang typed output
- **Error handling**: Graceful handling cho malformed messages
- **Field normalization**: Chuẩn hóa field names và data types
- **Omitempty support**: Handle optional fields correctly

**Lợi ích của Transform Layer:**

- **Decoupling**: Publishers và Clients độc lập về message format
- **Type safety**: Output luôn có structure chuẩn và type-safe
- **Validation**: Centralized validation và error handling
- **Evolution**: Dễ dàng thay đổi output format mà không ảnh hưởng publishers
- **Debugging**: Centralized logging và monitoring cho message processing

---

## Tổng Quan Kiến Trúc

### Kiến Trúc Hiện Tại

```
Dịch Vụ Publisher ──PUBLISH──► user_noti:userID ──► WebSocket ──► Tất Cả Kết Nối Người Dùng
```

### Kiến Trúc Mới

```
Dịch Vụ Publisher ──PUBLISH──► 2 Chủ Đề:
                                ├── project:projID:userID ──► Kết Nối Dự Án
                                └── job:jobID:userID ──► Kết Nối Công Việc

// Chủ đề người dùng tạm thời bỏ qua:
// ├── user_noti:userID ──► Kết Nối Chung
```

---

## Thiết Kế Chủ Đề & Cấu Trúc Tin Nhắn

<!-- ### Topic 1: User Notifications - COMMENTED OUT

**Pattern**: `user_noti:userID`
**Use Case**: General system notifications, alerts, updates

```go
// type UserNotificationMessage struct {
//     Title     string    `json:"title"`     // Notification title
//     Message   string    `json:"message"`   // Notification content
//     Priority  string    `json:"priority"`  // "low", "medium", "high", "urgent"
//     Category  string    `json:"category"`  // "system", "account", "security", "feature"
//     ActionURL string    `json:"action_url,omitempty"` // Optional action link
//     Timestamp time.Time `json:"timestamp"`
// }
```

**Example Message**:

```json
// {
//   "title": "System Maintenance",
//   "message": "Scheduled maintenance will occur at 2AM UTC",
//   "priority": "medium",
//   "category": "system",
//   "action_url": "/maintenance-details",
//   "timestamp": "2025-12-13T10:00:00Z"
// }
```

-->

### Chủ Đề 1: Thông Báo Dự Án

**Mẫu**: `project:projectID:userID`  
**Trường hợp sử dụng**: Tiến độ dự án, hoàn thành, lỗi

```go
// Project status enum
type ProjectStatus string

const (
    ProjectStatusProcessing ProjectStatus = "PROCESSING"  // Includes both crawling and analysis
    ProjectStatusCompleted  ProjectStatus = "COMPLETED"   // Project finished successfully
    ProjectStatusFailed     ProjectStatus = "FAILED"      // Project encountered fatal error
    ProjectStatusPaused     ProjectStatus = "PAUSED"      // Project temporarily stopped
)

type ProjectNotificationMessage struct {
    Status   ProjectStatus `json:"status"`             // Current project status (enum)
    Progress *Progress     `json:"progress,omitempty"` // Overall progress (omit if empty)
}

type Progress struct {
    Current    int      `json:"current"`     // Current completed items
    Total      int      `json:"total"`       // Total items to process
    Percentage float64  `json:"percentage"`  // Completion percentage (0-100)
    ETA        float64  `json:"eta"`         // Estimated time remaining in minutes
    Errors     []string `json:"errors"`      // Array of error messages encountered
}
```

#### Các Trường Thông Báo Dự Án

| Trường         | Ý Nghĩa Ngữ Nghĩa           | Nguồn Dữ Liệu                                                         | Mục Đích                                 | Giá Trị Ví Dụ                                   |
| -------------- | --------------------------- | --------------------------------------------------------------------- | ---------------------------------------- | ----------------------------------------------- |
| **`status`**   | Giai đoạn thực thi hiện tại | **Dịch vụ Dự án** - Cập nhật khi dự án tiến triển qua vòng đời        | Quản lý trạng thái UI, kích hoạt hành vi | `PROCESSING`, `COMPLETED`, `FAILED`, `PAUSED`   |
| **`progress`** | Tiến độ phân tích tổng thể  | **Dịch vụ Phân tích** - Cập nhật thời gian thực trong quá trình xử lý | Hiển thị tiến độ cho người dùng          | `{current: 800, total: 1000, percentage: 80.0}` |

#### **Phân Tích Các Trường Con Progress**

| Trường Con       | Ý Nghĩa Ngữ Nghĩa          | Nguồn Tính Toán                                              | Mục Đích                      | Ví Dụ                                                            |
| ---------------- | -------------------------- | ------------------------------------------------------------ | ----------------------------- | ---------------------------------------------------------------- |
| **`current`**    | Các mục đã hoàn thành      | **Bộ đếm dịch vụ** - Tăng dần khi các mục được xử lý         | Hiển thị tiến độ tuyệt đối    | `800` (trang đã thu thập)                                        |
| **`total`**      | Tổng số mục cần xử lý      | **Ước tính dịch vụ** - Đặt lúc bắt đầu, có thể được cập nhật | Hiển thị phạm vi công việc    | `1000` (tổng số trang)                                           |
| **`percentage`** | Phần trăm hoàn thành       | **Được tính**: `(current/total) * 100`                       | Thanh tiến độ trực quan       | `80.0` (hoàn thành 80%)                                          |
| **`eta`**        | Thời gian ước tính còn lại | **Được tính**: Dựa trên tiến độ hiện tại và metrics hệ thống | Quản lý kỳ vọng người dùng    | `8.5` (8.5 phút), `13.33` (13 phút 20 giây)                      |
| **`errors`**     | Mảng thông báo lỗi         | **Bộ thu thập lỗi dịch vụ** - Thu thập trong quá trình xử lý | Chi tiết lỗi, khắc phục sự cố | `["Không thể thu thập: timeout", "URL không hợp lệ: malformed"]` |

**Tin Nhắn Ví Dụ**:

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

#### Hành Vi Kích Hoạt UI Theo Trạng Thái

##### **`PROCESSING`** - Hiển Thị Giao Diện Tiến Độ

Khi nhận trạng thái PROCESSING, UI nên:

- Hiển thị thanh tiến độ phân tích
- Hiển thị metrics thời gian thực bao gồm tiến độ hiện tại, ETA và số lượng lỗi
- Giữ người dùng ở trang hiện tại với cập nhật tiến độ trực tiếp

##### **`COMPLETED`** - Chuyển Hướng Đến Kết Quả

Khi nhận trạng thái COMPLETED, UI nên:

- Hiển thị thông báo thành công rằng dự án đã hoàn thành
- Ẩn tất cả các phần tử giao diện tiến độ
- Tự động chuyển hướng đến trang kết quả/dữ liệu dự án sau một khoảng thời gian ngắn
- Cho phép người dùng xem thông báo hoàn thành trước khi chuyển hướng

##### **`FAILED`** - Hiển Thị Giao Diện Lỗi

Khi nhận trạng thái FAILED, UI nên:

- Ẩn các phần tử giao diện tiến độ
- Hiển thị trạng thái lỗi với tên dự án và thông báo lỗi
- Cung cấp các nút hành động để thử lại, xem logs và liên hệ hỗ trợ
- Hiển thị tùy chọn kết quả một phần nếu có dữ liệu được thu thập trong quá trình xử lý

##### **`PAUSED`** - Hiển Thị Trạng Thái Tạm Dừng

Khi nhận trạng thái PAUSED, UI nên:

- Hiển thị chỉ báo tạm dừng với biểu tượng và kiểu dáng phù hợp
- Đóng băng thanh tiến độ ở trạng thái hiện tại mà không cập nhật thêm
- Hiển thị các nút hành động tiếp tục và hủy bỏ
- Dừng cập nhật tiến độ trực tiếp
- Hiển thị lý do tạm dừng nếu có

### Chủ Đề 2: Thông Báo Công Việc

**Mẫu**: `job:jobID:userID`  
**Trường hợp sử dụng**: Kết quả batch công việc thu thập mạng xã hội, cập nhật tiến độ

**Logic Kinh Doanh**: Mỗi job sẽ có nhiều messages - mỗi lần crawl xong 1 batch data sẽ publish 1 message để UI update real-time.

```go
type JobNotificationMessage struct {
    Platform Platform    `json:"platform"`           // Social media platform enum
    Status   JobStatus   `json:"status"`             // Current job processing status
    Batch    *BatchData  `json:"batch,omitempty"`    // Current batch crawl results (omit if empty)
    Progress *Progress   `json:"progress,omitempty"` // Overall job progress statistics (omit if empty)
}

// Platform enum
type Platform string

const (
    PlatformTikTok    Platform = "TIKTOK"    // TikTok platform
    PlatformYouTube   Platform = "YOUTUBE"   // YouTube platform
    PlatformInstagram Platform = "INSTAGRAM" // Instagram platform
)

// Job status enum (aligned with ProjectStatus)
type JobStatus string

const (
    JobStatusProcessing JobStatus = "PROCESSING" // Job is actively crawling/processing
    JobStatusCompleted  JobStatus = "COMPLETED"  // Job finished all batches
    JobStatusFailed     JobStatus = "FAILED"     // Job encountered fatal error
    JobStatusPaused     JobStatus = "PAUSED"     // Job temporarily stopped
)

// BatchData - Results from a single crawl batch
type BatchData struct {
    Keyword     string         `json:"keyword"`      // Search keyword for this batch
    ContentList []ContentItem  `json:"content_list"` // Crawled content items
    CrawledAt   string         `json:"crawled_at"`   // When this batch was processed (ISO timestamp)
}

type Progress struct {
    Current    int      `json:"current"`     // Current completed items
    Total      int      `json:"total"`       // Total items to process
    Percentage float64  `json:"percentage"`  // Completion percentage (0-100)
    ETA        float64  `json:"eta"`         // Estimated time remaining in minutes
    Errors     []string `json:"errors"`      // Array of error messages encountered
}

// ContentItem - Single social media content (simplified for UI)
type ContentItem struct {
    ID          string            `json:"id"`          // Content unique ID
    Text        string            `json:"text"`        // Content text/caption
    Author      AuthorInfo        `json:"author"`      // Author information
    Metrics     EngagementMetrics `json:"metrics"`     // Engagement statistics
    Media       *MediaInfo        `json:"media,omitempty"`       // Media information (if any)
    PublishedAt string            `json:"published_at"` // When content was published (ISO timestamp)
    Permalink   string            `json:"permalink"`   // Direct link to content
}

// AuthorInfo - Content author details
type AuthorInfo struct {
    ID         string `json:"id"`         // Author unique ID
    Username   string `json:"username"`   // Author username/handle
    Name       string `json:"name"`       // Author display name
    Followers  int    `json:"followers"`  // Follower count
    IsVerified bool   `json:"is_verified"` // Verification status
    AvatarURL  string `json:"avatar_url"` // Profile picture URL
}

// EngagementMetrics - Content engagement statistics
type EngagementMetrics struct {
    Views    int     `json:"views"`    // View count
    Likes    int     `json:"likes"`    // Like count
    Comments int     `json:"comments"` // Comment count
    Shares   int     `json:"shares"`   // Share count
    Rate     float64 `json:"rate"`     // Engagement rate percentage
}

// MediaInfo - Media content details
type MediaInfo struct {
    Type      string `json:"type"`       // "video", "image", "audio"
    Duration  int    `json:"duration,omitempty"`   // Duration in seconds (for video/audio)
    Thumbnail string `json:"thumbnail"`  // Thumbnail/preview URL
    URL       string `json:"url"`        // Media file URL
}
```

**Example Messages**:

#### **Batch Result Message** (Real-time crawl updates)

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
    "crawled_at": "2024-12-13T10:15:30Z"
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

#### **Job Completion Message** (Final status)

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

### Các Trường Thông Báo Công Việc

| Trường         | Ý Nghĩa Ngữ Nghĩa                  | Nguồn Dữ Liệu                                                 | Mục Đích                                   | Giá Trị Ví Dụ                                           |
| -------------- | ---------------------------------- | ------------------------------------------------------------- | ------------------------------------------ | ------------------------------------------------------- |
| **`platform`** | Nền tảng mạng xã hội được thu thập | **Dịch vụ Job** - Đặt dựa trên cấu hình job                   | Hiển thị trong UI, phân loại theo nền tảng | `TIKTOK`, `YOUTUBE`, `INSTAGRAM`                        |
| **`status`**   | Trạng thái thực thi job hiện tại   | **Dịch vụ Crawler** - Cập nhật khi job tiến triển qua batches | Quản lý trạng thái UI, kích hoạt hành vi   | `PROCESSING`, `COMPLETED`, `FAILED`, `PAUSED`           |
| **`batch`**    | Kết quả batch crawl hiện tại       | **Dịch vụ Crawler** - Dữ liệu hoàn thành batch thời gian thực | Hiển thị nội dung crawl mới nhất cho user  | `{keyword: "christmas trends", content_list: [...]}`    |
| **`progress`** | Thống kê tiến độ job tổng thể      | **Dịch vụ Job** - Tổng hợp từ tất cả batches đã hoàn thành    | Hiển thị tiến độ và hiệu suất job          | `{current: 15, total: 50, percentage: 30.0, eta: 25.5}` |

##### **Phân Tích Các Trường Con Batch**

| Trường Con         | Ý Nghĩa Ngữ Nghĩa              | Nguồn Tính Toán                                           | Mục Đích                           | Ví Dụ                                       |
| ------------------ | ------------------------------ | --------------------------------------------------------- | ---------------------------------- | ------------------------------------------- |
| **`keyword`**      | Từ khóa tìm kiếm cho batch này | **Dịch vụ Crawler** - Từ cấu hình từ khóa job             | Hiển thị những gì đang được tìm    | "christmas trends", "tech reviews"          |
| **`content_list`** | Mảng nội dung đã crawl         | **Dịch vụ Crawler** - Nội dung được phân tích và cấu trúc | Hiển thị kết quả crawl cho user    | `[{id: "123", text: "...", author: {...}}]` |
| **`crawled_at`**   | Timestamp hoàn thành batch     | **Dịch vụ Crawler** - Khi batch hoàn thành xử lý          | Thời gian batch, thông tin độ tươi | "2024-12-13T10:15:30Z"                      |

##### **Phân Tích Các Trường Con ContentItem**

| Trường Con         | Ý Nghĩa Ngữ Nghĩa                | Nguồn Dữ Liệu                                       | Mục Đích                            | Ví Dụ                                                   |
| ------------------ | -------------------------------- | --------------------------------------------------- | ----------------------------------- | ------------------------------------------------------- |
| **`id`**           | ID nội dung cụ thể theo nền tảng | **Platform API** - Định danh duy nhất từ nền tảng   | Khử trùng nội dung, liên kết        | "7312345678901234567" (TikTok), "dQw4w9WgXcQ" (YouTube) |
| **`text`**         | Chú thích/mô tả nội dung         | **Platform API** - Văn bản nội dung từ nền tảng     | Xem trước nội dung, tìm kiếm        | "Christmas decoration ideas! 🎄✨ #christmas"           |
| **`author`**       | Thông tin người tạo nội dung     | **Platform API** - Dữ liệu hồ sơ tác giả            | Phân tích tác giả, ID influencer    | `{username: "@decorqueen", followers: 125000}`          |
| **`metrics`**      | Thống kê tương tác               | **Platform API** - Dữ liệu tương tác thời gian thực | Phân tích hiệu suất, xu hướng       | `{views: 2500000, likes: 180000, rate: 7.88}`           |
| **`media`**        | Thông tin tệp media              | **Platform API** - Metadata và URLs media           | Phân tích media, liên kết tải xuống | `{type: "video", duration: 45, url: "..."}`             |
| **`published_at`** | Thời gian xuất bản nội dung      | **Platform API** - Khi nội dung được đăng ban đầu   | Độ tươi nội dung, dòng thời gian    | "2024-12-10T15:30:00Z"                                  |
| **`permalink`**    | Liên kết trực tiếp đến nội dung  | **Platform API** - URL chính tắc đến nội dung       | Truy cập bên ngoài, xác minh        | "https://tiktok.com/@user/video/123"                    |

##### **Phân Tích Các Trường Con Progress (Tái Sử Dụng Từ Project)**

Job notifications sử dụng cùng struct Progress như Project notifications để đảm bảo tính nhất quán:

| Trường Con       | Ý Nghĩa Ngữ Nghĩa Cho Job   | Nguồn Tính Toán                                   | Mục Đích                      | Ví Dụ                                        |
| ---------------- | --------------------------- | ------------------------------------------------- | ----------------------------- | -------------------------------------------- |
| **`current`**    | Số batches đã hoàn thành    | **Dịch vụ Job** - Tăng dần khi batches hoàn thành | Chỉ báo tiến độ tuyệt đối     | `15` (15 trong số 50 batches đã xong)        |
| **`total`**      | Tổng số batches cần xử lý   | **Dịch vụ Job** - Tính từ cấu hình job            | Hiển thị phạm vi công việc    | `50` (50 keyword batches cần xử lý)          |
| **`percentage`** | Phần trăm hoàn thành        | **Được tính**: `(current/total) * 100`            | Thanh tiến độ trực quan       | `30.0` (hoàn thành 30%)                      |
| **`eta`**        | Thời gian ước tính còn lại  | **Dịch vụ Job** - Tính từ tốc độ hoàn thành batch | Quản lý kỳ vọng người dùng    | `25.5` (còn 25.5 phút)                       |
| **`errors`**     | Mảng thông báo lỗi tổng hợp | **Dịch vụ Job** - Tổng hợp lỗi từ tất cả batches  | Chi tiết lỗi, khắc phục sự cố | `["Rate limit exceeded", "Network timeout"]` |

#### Hành Vi Kích Hoạt UI Theo Trạng Thái

##### **`PROCESSING`** - Hiển Thị Giao Diện Tiến Độ

Khi nhận trạng thái PROCESSING, UI nên:

- Hiển thị thanh tiến độ cho job crawling
- Hiển thị metrics thời gian thực bao gồm tiến độ hiện tại, ETA và số lượng lỗi tổng hợp
- Append nội dung mới từ batch vào danh sách tổng hợp
- Hiển thị keyword đang được xử lý và lỗi tổng hợp từ progress

##### **`COMPLETED`** - Chuyển Hướng Đến Kết Quả

Khi nhận trạng thái COMPLETED, UI nên:

- Hiển thị thông báo thành công rằng job đã hoàn thành
- Ẩn tất cả các phần tử giao diện tiến độ
- Hiển thị tổng số nội dung đã thu thập từ tất cả batches
- Cung cấp tùy chọn xem, tải xuống hoặc phân tích dữ liệu

##### **`FAILED`** - Hiển Thị Giao Diện Lỗi

Khi nhận trạng thái FAILED, UI nên:

- Ẩn các phần tử giao diện tiến độ
- Hiển thị trạng thái lỗi với thông báo lỗi chi tiết
- Cung cấp các nút hành động để thử lại, xem logs và liên hệ hỗ trợ
- Hiển thị kết quả một phần nếu có dữ liệu được thu thập trước khi lỗi

##### **`PAUSED`** - Hiển Thị Trạng Thái Tạm Dừng

Khi nhận trạng thái PAUSED, UI nên:

- Hiển thị chỉ báo tạm dừng với biểu tượng và kiểu dáng phù hợp
- Đóng băng thanh tiến độ ở trạng thái hiện tại mà không cập nhật thêm
- Hiển thị các nút hành động tiếp tục và hủy bỏ
- Dừng việc append nội dung mới từ batches

### **UI Data Handling**

#### **Job Notification Flow**

- **Multiple Messages per Job**: Mỗi batch hoàn thành kích hoạt một message
- **Real-time Content Feed**: UI append content mới từ batch vào feed hiện có
- **Progress Aggregation**: Thống kê tổng hợp cập nhật với mỗi batch
- **Error Resilience**: Lỗi batch riêng lẻ không làm fail toàn bộ job

#### **Field Usage Guidelines**

**Frontend UI:**

- **`platform`**: Hiển thị icon nền tảng, phân loại job
- **`status`**: UI state management (spinner, checkmark, error)
- **`progress`**: Thanh tiến độ tổng thể và ETA
- **`batch.content_list`**: Append content mới vào feed
- **`batch.keyword`**: Hiển thị keyword đang xử lý
- **`progress.errors`**: Hiển thị lỗi tổng hợp với retry options

**Backend Services:**

- **Job Service**: Tạo jobs, quản lý lifecycle, tổng hợp statistics
- **Crawler Service**: Xử lý batches, crawl content, publish results

---

## Client Connection URLs

```javascript
// Project: ws://localhost:8081/ws?projectId=proj_123
// Job: ws://localhost:8081/ws?jobId=job_789
// General: ws://localhost:8081/ws
```

---

## Redis Implementation

**Multi-Pattern Subscription**: `project:*`, `job:*`

### 2. Message Handler with Transform Layer

```go
func (s *Subscriber) handleMessage(channel string, payload string) {
    parts := strings.Split(channel, ":")

    switch parts[0] {
    case "project":
        projectID, userID := parts[1], parts[2]
        // MANDATORY: Transform Redis input to standardized output
        standardMsg, err := s.transformProjectNotification(payload, projectID, userID)
        if err != nil {
            s.logger.Errorf(s.ctx, "CRITICAL: Transform failed: %v", err)
            return
        }
        s.handleProjectNotification(standardMsg)

    case "job":
        jobID, userID := parts[1], parts[2]
        // MANDATORY: Transform Redis input to standardized output
        standardMsg, err := s.transformJobNotification(payload, jobID, userID)
        if err != nil {
            s.logger.Errorf(s.ctx, "CRITICAL: Transform failed: %v", err)
            return
        }
        s.handleJobNotification(standardMsg)
    }
}
```

### 3. Transform Functions (MANDATORY)

```go
// Transform Project Input → Output
func (s *Subscriber) transformProjectNotification(payload, projectID, userID string) (*ProjectNotificationMessage, error) {
    var inputMsg ProjectInputMessage
    if err := json.Unmarshal([]byte(payload), &inputMsg); err != nil {
        return nil, fmt.Errorf("parse failed: %w", err)
    }

    standardMsg := &ProjectNotificationMessage{
        Status: ProjectStatus(inputMsg.Status),
    }

    if inputMsg.Progress != nil {
        standardMsg.Progress = &Progress{
            Current: inputMsg.Progress.Current,
            Total: inputMsg.Progress.Total,
            Percentage: inputMsg.Progress.Percentage,
            ETA: inputMsg.Progress.ETA,
            Errors: inputMsg.Progress.Errors,
        }
    }

    return standardMsg, nil
}

// Transform Job Input → Output
func (s *Subscriber) transformJobNotification(payload, jobID, userID string) (*JobNotificationMessage, error) {
    var inputMsg JobInputMessage
    if err := json.Unmarshal([]byte(payload), &inputMsg); err != nil {
        return nil, fmt.Errorf("parse failed: %w", err)
    }

    standardMsg := &JobNotificationMessage{
        Platform: Platform(inputMsg.Platform),
        Status:   JobStatus(inputMsg.Status),
    }

    if inputMsg.Batch != nil {
        standardMsg.Batch = &BatchData{
            Keyword:     inputMsg.Batch.Keyword,
            ContentList: s.transformContentList(inputMsg.Batch.ContentList),
            CrawledAt:   inputMsg.Batch.CrawledAt,
        }
    }

    if inputMsg.Progress != nil {
        standardMsg.Progress = &Progress{
            Current: inputMsg.Progress.Current,
            Total: inputMsg.Progress.Total,
            Percentage: inputMsg.Progress.Percentage,
            ETA: inputMsg.Progress.ETA,
            Errors: inputMsg.Progress.Errors,
        }
    }

    return standardMsg, nil
}

// MANDATORY Transform Layer Input Types (must match redis_publisher_input_specification.md)
type ProjectInputMessage struct {
    Status   string         `json:"status"`
    Progress *ProgressInput `json:"progress,omitempty"`
}

type JobInputMessage struct {
    Platform string         `json:"platform"`
    Status   string         `json:"status"`
    Batch    *BatchInput    `json:"batch,omitempty"`
    Progress *ProgressInput `json:"progress,omitempty"`
}

type ProgressInput struct {
    Current    int      `json:"current"`
    Total      int      `json:"total"`
    Percentage float64  `json:"percentage"`
    ETA        float64  `json:"eta"`
    Errors     []string `json:"errors"`
}

type BatchInput struct {
    Keyword     string         `json:"keyword"`
    ContentList []ContentInput `json:"content_list"`
    CrawledAt   string         `json:"crawled_at"`
}

type ContentInput struct {
    ID          string       `json:"id"`
    Text        string       `json:"text"`
    Author      AuthorInput  `json:"author"`
    Metrics     MetricsInput `json:"metrics"`
    Media       *MediaInput  `json:"media,omitempty"`
    PublishedAt string       `json:"published_at"`
    Permalink   string       `json:"permalink"`
}

type AuthorInput struct {
    ID         string `json:"id"`
    Username   string `json:"username"`
    Name       string `json:"name"`
    Followers  int    `json:"followers"`
    IsVerified bool   `json:"is_verified"`
    AvatarURL  string `json:"avatar_url"`
}

type MetricsInput struct {
    Views    int     `json:"views"`
    Likes    int     `json:"likes"`
    Comments int     `json:"comments"`
    Shares   int     `json:"shares"`
    Rate     float64 `json:"rate"`
}

type MediaInput struct {
    Type      string `json:"type"`
    Duration  int    `json:"duration,omitempty"`
    Thumbnail string `json:"thumbnail"`
    URL       string `json:"url"`
}
```

### 3. Hub Extensions

#### Connection Structure Update

```go
type Connection struct {
    // Existing fields
    hub        *Hub
    conn       *websocket.Conn
    userID     string
    send       chan []byte

    // NEW: Subscription filters
    projectID  string  // Empty if not subscribed to project
    jobID      string  // Empty if not subscribed to job

    // Existing fields
    pongWait   time.Duration
    pingPeriod time.Duration
    writeWait  time.Duration
    logger     log.Logger
    done       chan struct{}
}
```

#### New Hub Methods

```go
// Existing method
func (h *Hub) SendToUser(userID string, message *Message)

// NEW: Project-specific sending
func (h *Hub) SendToUserWithProject(userID, projectID string, message *Message) {
    h.mu.RLock()
    connections := h.connections[userID]
    h.mu.RUnlock()

    if len(connections) == 0 {
        return
    }

    data, err := message.ToJSON()
    if err != nil {
        h.logger.Errorf(context.Background(), "Failed to marshal message: %v", err)
        return
    }

    // Send only to connections subscribed to this project
    sentCount := 0
    for _, conn := range connections {
        if conn.projectID == projectID {
            select {
            case conn.send <- data:
                sentCount++
            default:
                h.logger.Warnf(context.Background(), "Failed to send message to user %s (buffer full)", userID)
            }
        }
    }

    h.totalMessagesSent.Add(int64(sentCount))
}

// NEW: Job-specific sending
func (h *Hub) SendToUserWithJob(userID, jobID string, message *Message) {
    h.mu.RLock()
    connections := h.connections[userID]
    h.mu.RUnlock()

    if len(connections) == 0 {
        return
    }

    data, err := message.ToJSON()
    if err != nil {
        h.logger.Errorf(context.Background(), "Failed to marshal message: %v", err)
        return
    }

    // Send only to connections subscribed to this job
    sentCount := 0
    for _, conn := range connections {
        if conn.jobID == jobID {
            select {
            case conn.send <- data:
                sentCount++
            default:
                h.logger.Warnf(context.Background(), "Failed to send message to user %s (buffer full)", userID)
            }
        }
    }

    h.totalMessagesSent.Add(int64(sentCount))
}
```

### 4. Handler Parameter Parsing

#### Updated WebSocket Handler

```go
func (h *Handler) HandleWebSocket(c *gin.Context) {
    // HttpOnly Cookie Authentication ONLY (no token fallback)
    token, err := c.Cookie(h.cookieConfig.Name)
    if err != nil || token == "" {
        h.logger.Warn(context.Background(), "WebSocket connection rejected: missing auth cookie")
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication cookie"})
        return
    }

    // Validate JWT from cookie
    userID, err := h.jwtValidator.ExtractUserID(token)
    if err != nil {
        h.logger.Warnf(context.Background(), "WebSocket connection rejected: invalid token - %v", err)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
        return
    }

    // NEW: Parse subscription parameters
    projectID := c.Query("projectId")
    jobID := c.Query("jobId")

    // Validate access permissions
    if projectID != "" {
        if !h.authService.CanAccessProject(userID, projectID) {
            c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized access to project"})
            return
        }
    }

    if jobID != "" {
        if !h.authService.CanAccessJob(userID, jobID) {
            c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized access to job"})
            return
        }
    }

    // Existing WebSocket upgrade...
    conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        h.logger.Errorf(context.Background(), "Failed to upgrade connection: %v", err)
        return
    }

    // NEW: Create connection with filters
    connection := NewConnectionWithFilters(
        h.hub,
        conn,
        userID,
        projectID,  // NEW
        jobID,      // NEW
        h.wsConfig.PongWait,
        h.wsConfig.PingPeriod,
        h.wsConfig.WriteWait,
        h.logger,
    )

    // Register and start connection
    h.hub.register <- connection
    connection.Start()

    h.logger.Infof(context.Background(),
        "WebSocket connection established for user: %s, project: %s, job: %s",
        userID, projectID, jobID)
}
```

---

## Triển Khai Phía Publisher

**⚠️ Publishers giữ nguyên format hiện tại** - Không cần thay đổi code publisher. WebSocket service sẽ transform messages.

**Publisher Libraries**: Tạo ProjectNotificationPublisher, JobNotificationPublisher với methods publish theo channels `project:projectID:userID` và `job:jobID:userID`

---

## Ví Dụ Sử Dụng

**Project Tracking**: Frontend connect với `?projectId=proj_123`, nhận project progress/completion messages

**Job Monitoring**: Frontend connect với `?jobId=job_789`, nhận job started/progress/completed/failed messages

**Backend**: Publishers sử dụng channels `project:projectID:userID` và `job:jobID:userID` để publish messages theo existing format

---

## Chiến Lược Di Chuyển

**5 Giai Đoạn (6 tuần)**:

1. **WebSocket Service** (1-2 tuần): **MANDATORY Transform Layer**, Hub filtering, Connection filters, Handler params
2. **Publisher Libraries** (2-3 tuần): ProjectNotificationPublisher, JobNotificationPublisher với dual publishing
3. **Service Integration** (3-4 tuần): Update Project/Job services, feature flags
4. **Frontend Update** (4-5 tuần): Dashboard với projectId/jobId params, gradual rollout
5. **Cleanup** (tuần 6): Remove dual publishing, optimize performance

---

## 🔒 Cân Nhắc Bảo Mật

**Authorization**: Validate user access to projectID/jobID before connection

```go
func (h *Handler) validateAccess(userID, projectID, jobID string) error {
    if projectID != "" {
        if !h.authService.CanAccessProject(userID, projectID) {
            return fmt.Errorf("unauthorized access to project: %s", projectID)
        }
    }

    if jobID != "" {
        if !h.authService.CanAccessJob(userID, jobID) {
            return fmt.Errorf("unauthorized access to job: %s", jobID)
        }
    }

    return nil
}
```

### 2. Rate Limiting

```go
type FilterLimits struct {
    MaxProjectConnections int `env:"WS_MAX_PROJECT_CONNECTIONS" envDefault:"5"`
    MaxJobConnections     int `env:"WS_MAX_JOB_CONNECTIONS" envDefault:"3"`
}
```

### 3. Input Validation

```go
func validateProjectID(projectID string) error {
    if len(projectID) == 0 || len(projectID) > 50 {
        return fmt.Errorf("invalid project ID length")
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(projectID) {
        return fmt.Errorf("invalid project ID format")
    }
    return nil
}
```

---

## Phân Tích Tác Động Hiệu Suất

**Memory**: +25% per connection (thêm projectID/jobID fields)
**CPU**: +2-5ms message filtering, <5% tổng thể
**Network**: 60-80% giảm messages không cần thiết → Net positive
**Redis**: +2 PSUBSCRIBE patterns, minimal impact

---

## Chiến Lược Kiểm Thử

**Unit Tests**: Publisher message structure, Subscriber parsing, Hub filtering
**Integration Tests**: End-to-end project/job notifications với filtering
**Load Tests**: 1000 connections với mixed filters, measure memory/CPU/Redis performance

---

## Cập Nhật Cấu Hình

**Environment Variables**: Filter limits (max connections), feature flags (enable filtering), performance tuning (cache size, buffer size)

---

## Chỉ Số Thành Công

**Functional**: Targeted message delivery, authorization working, backward compatibility
**Performance**: Memory <30% increase, filtering <5ms latency, connection <200ms
**Business**: 60-80% giảm unnecessary messages, better UX, faster page loads
