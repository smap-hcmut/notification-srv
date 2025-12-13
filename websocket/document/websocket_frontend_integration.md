# WebSocket Frontend Integration - Đặc Tả UI Behavior

## Tổng Quan

Tài liệu này cung cấp **đặc tả chi tiết về UI behavior** cho Frontend Developers để implement WebSocket integration với topic-based subscription system. Document này định nghĩa cách xử lý messages, cập nhật UI states, và quản lý user experience.

---

## WebSocket Connection Setup

### 1. Connection URLs

```javascript
// Project-specific connection
const projectWs = new WebSocket(
  `ws://localhost:8081/ws?projectId=${projectId}`
);

// Job-specific connection
const jobWs = new WebSocket(`ws://localhost:8081/ws?jobId=${jobId}`);

// General connection (backward compatibility)
const generalWs = new WebSocket("ws://localhost:8081/ws");
```

### 2. Authentication

- **HttpOnly Cookie Only** - Không sử dụng JWT token fallback
- **Automatic Authentication** - Cookie được gửi tự động với WebSocket request
- **Authorization Check** - Server validate quyền truy cập project/job

---

## Project Notifications UI Behavior

### 1. Message Structure

```typescript
interface ProjectNotificationMessage {
  status: ProjectStatus; // Current project status (enum)
  progress?: Progress; // Overall progress (omit if empty)
}

enum ProjectStatus {
  PROCESSING = "PROCESSING", // Includes both crawling and analysis
  COMPLETED = "COMPLETED", // Project finished successfully
  FAILED = "FAILED", // Project encountered fatal error
  PAUSED = "PAUSED", // Project temporarily stopped
}

interface Progress {
  current: number; // Current completed items
  total: number; // Total items to process
  percentage: number; // Completion percentage (0-100)
  eta: number; // Estimated time remaining in minutes
  errors: string[]; // Array of error messages encountered
}
```

### 2. Status-Specific UI Behaviors

#### **PROCESSING Status**

Khi nhận trạng thái PROCESSING, UI nên thực hiện theo thứ tự:

1. **Hiển thị giao diện tiến độ** - Bật progress bar và metrics display
2. **Cập nhật status indicator** - Hiển thị trạng thái "Đang xử lý" với icon loading và màu sắc processing
3. **Cập nhật thanh tiến độ** - Nếu có progress, hiển thị progress bar với label "Đang phân tích"
4. **Hiển thị ETA** - Hiển thị thời gian còn lại từ progress.eta
5. **Cập nhật danh sách lỗi** - Hiển thị errors từ progress
6. **Hiển thị action buttons** - Bật các nút "Tạm dừng" và "Hủy bỏ"
7. **Kích hoạt live updates** - Giữ người dùng ở trang hiện tại với cập nhật real-time

#### **COMPLETED Status**

Khi nhận trạng thái COMPLETED, UI nên thực hiện theo thứ tự:

1. **Ẩn giao diện tiến độ** - Tắt tất cả progress bars và metrics display
2. **Cập nhật status indicator** - Hiển thị trạng thái "Hoàn thành" với icon success và màu xanh
3. **Hiển thị thông báo hoàn thành** - Popup notification với tên project và thông báo thành công
4. **Ẩn ETA display** - Không còn cần hiển thị thời gian còn lại
5. **Xóa danh sách lỗi** - Clear tất cả error messages
6. **Hiển thị action buttons kết quả** - Bật các nút "Xem kết quả", "Tải xuống", "Dự án mới"
7. **Lên lịch chuyển hướng tự động** - Hiển thị countdown 5 giây trước khi chuyển đến trang kết quả
8. **Cho phép hủy chuyển hướng** - Cung cấp nút để người dùng có thể hủy auto-redirect

#### **FAILED Status**

Khi nhận trạng thái FAILED, UI nên thực hiện theo thứ tự:

1. **Ẩn giao diện tiến độ** - Tắt tất cả progress bars và metrics display
2. **Cập nhật status indicator** - Hiển thị trạng thái "Thất bại" với icon error và màu đỏ
3. **Hiển thị thông báo lỗi** - Popup notification với tên project và thông báo lỗi (không tự ẩn)
4. **Ẩn ETA display** - Không còn cần hiển thị thời gian còn lại
5. **Hiển thị chi tiết lỗi** - Hiển thị errors từ progress
6. **Hiển thị action buttons lỗi** - Bật các nút "Thử lại", "Xem logs", "Liên hệ hỗ trợ", "Kết quả một phần"
7. **Kiểm tra kết quả một phần** - Nếu có current > 0 trong progress, hiển thị thông báo có dữ liệu một phần
8. **Kích hoạt nút kết quả một phần** - Nếu có partial results, enable nút để xem dữ liệu đã thu thập

#### **PAUSED Status**

Khi nhận trạng thái PAUSED, UI nên thực hiện theo thứ tự:

1. **Hiển thị giao diện tạm dừng** - Chuyển sang chế độ paused với styling đặc biệt
2. **Cập nhật status indicator** - Hiển thị trạng thái "Tạm dừng" với icon pause và màu vàng
3. **Đóng băng progress bars** - Giữ nguyên progress hiện tại nhưng thêm styling "frozen"
4. **Ẩn ETA display** - Không còn tính toán thời gian còn lại
5. **Hiển thị lý do tạm dừng** - Nếu có pause_reason trong message, hiển thị lý do
6. **Hiển thị action buttons tạm dừng** - Bật các nút "Tiếp tục" và "Hủy bỏ"
7. **Tắt live updates** - Dừng việc cập nhật real-time
8. **Thêm visual indicators** - Hiển thị các indicator cho biết trạng thái tạm dừng

### 3. Project Field Update Behaviors

Khi nhận message mới cho Project, các field được xử lý như sau:

| Field                     | Update Behavior | Mô tả                                                   |
| ------------------------- | --------------- | ------------------------------------------------------- |
| **`status`**              | **OVERRIDE**    | Luôn cập nhật trạng thái mới, kích hoạt UI state change |
| **`progress`**            | **OVERRIDE**    | Thay thế toàn bộ object progress (omit if empty)        |
| **`progress.current`**    | **OVERRIDE**    | Thay thế số lượng hiện tại (không cộng dồn)             |
| **`progress.total`**      | **OVERRIDE**    | Cập nhật tổng số (có thể thay đổi trong quá trình)      |
| **`progress.percentage`** | **OVERRIDE**    | Tính toán lại từ current/total                          |
| **`progress.eta`**        | **OVERRIDE**    | Cập nhật ETA mới nhất (float64 minutes)                 |
| **`progress.errors`**     | **OVERRIDE**    | Thay thế toàn bộ danh sách lỗi (không append)           |

**Lưu ý quan trọng:**

- **Omit empty behavior** - Nếu không có progress thì field đó không được gửi
- **Không append errors** - Mỗi message chứa danh sách lỗi đầy đủ tại thời điểm đó
- **Progress không cộng dồn** - current là số tuyệt đối, không phải increment
- **ETA là float64** - Thời gian tính bằng phút, có thể có số thập phân (8.5 = 8 phút 30 giây)

---

## Job Notifications UI Behavior

### 1. Message Structure

```typescript
interface JobNotificationMessage {
  platform: Platform; // Social media platform enum
  status: JobStatus; // Current job processing status
  batch?: BatchData; // Current batch crawl results (omit if empty)
  progress?: Progress; // Overall job progress statistics (omit if empty)
}

enum Platform {
  TIKTOK = "TIKTOK", // TikTok platform
  YOUTUBE = "YOUTUBE", // YouTube platform
  INSTAGRAM = "INSTAGRAM", // Instagram platform
}

enum JobStatus {
  PROCESSING = "PROCESSING", // Job is actively crawling/processing
  COMPLETED = "COMPLETED", // Job finished all batches
  FAILED = "FAILED", // Job encountered fatal error
  PAUSED = "PAUSED", // Job temporarily stopped
}

interface BatchData {
  keyword: string; // Search keyword for this batch
  content_list: ContentItem[]; // Crawled content items
  crawled_at: string; // When this batch was processed (ISO timestamp)
}

interface ContentItem {
  id: string; // Content unique ID
  text: string; // Content text/caption
  author: AuthorInfo; // Author information
  metrics: EngagementMetrics; // Engagement statistics
  media?: MediaInfo; // Media information (if any)
  published_at: string; // When content was published (ISO timestamp)
  permalink: string; // Direct link to content
}

interface AuthorInfo {
  id: string; // Author unique ID
  username: string; // Author username/handle
  name: string; // Author display name
  followers: number; // Follower count
  is_verified: boolean; // Verification status
  avatar_url: string; // Profile picture URL
}

interface EngagementMetrics {
  views: number; // View count
  likes: number; // Like count
  comments: number; // Comment count
  shares: number; // Share count
  rate: number; // Engagement rate percentage
}

interface MediaInfo {
  type: string; // "video", "image", "audio"
  duration?: number; // Duration in seconds (for video/audio)
  thumbnail: string; // Thumbnail/preview URL
  url: string; // Media file URL
}
```

### 2. Status-Specific UI Behaviors

#### **PROCESSING Status**

Khi nhận trạng thái PROCESSING, UI nên thực hiện theo thứ tự:

1. **Hiển thị giao diện job progress** - Bật job progress bar và platform indicator
2. **Cập nhật status indicator** - Hiển thị "Đang thu thập" với icon loading và màu processing
3. **Cập nhật platform indicator** - Hiển thị logo/icon của platform (TikTok/YouTube/Instagram)
4. **Xử lý batch data** - Nếu có batch, append content mới vào feed và update keyword hiện tại
5. **Cập nhật job progress** - Hiển thị tiến độ tổng thể từ progress field
6. **Hiển thị ETA** - Hiển thị thời gian còn lại cho toàn bộ job
7. **Cập nhật error aggregation** - Hiển thị tổng hợp lỗi từ tất cả batches
8. **Hiển thị action buttons** - Bật các nút "Tạm dừng" và "Hủy bỏ"
9. **Kích hoạt content feed updates** - Enable real-time content feed với animation

#### **COMPLETED Status**

Khi nhận trạng thái COMPLETED, UI nên thực hiện theo thứ tự:

1. **Ẩn job progress interface** - Tắt progress bar và ETA display
2. **Cập nhật status indicator** - Hiển thị "Hoàn thành" với icon success và màu xanh
3. **Hiển thị completion summary** - Popup với tổng số content, batches thành công, platform
4. **Clear current keyword** - Hiển thị "Hoàn thành tất cả từ khóa"
5. **Hiển thị final statistics** - Thống kê chi tiết về engagement, top performers, content types
6. **Hiển thị action buttons hoàn thành** - Bật các nút "Xem tất cả", "Tải xuống", "Phân tích", "Job mới"
7. **Tắt real-time updates** - Dừng việc cập nhật content feed
8. **Hiển thị job summary** - Metrics tổng hợp về performance và kết quả

#### **FAILED Status**

Khi nhận trạng thái FAILED, UI nên thực hiện theo thứ tự:

1. **Ẩn job progress interface** - Tắt progress bar và ETA display
2. **Cập nhật status indicator** - Hiển thị "Thất bại" với icon error và màu đỏ
3. **Hiển thị error summary** - Popup với số content thu thập được, batches hoàn thành, số lỗi
4. **Hiển thị chi tiết lỗi** - Danh sách lỗi chi tiết từ progress.errors
5. **Kiểm tra partial results** - Nếu có content đã thu thập, hiển thị thông báo và nút xem
6. **Hiển thị action buttons lỗi** - Bật các nút "Thử lại", "Xem logs", "Hỗ trợ", "Kết quả một phần"
7. **Tắt content feed updates** - Dừng việc append content mới
8. **Hiển thị failure statistics** - Thống kê về lỗi và partial results

#### **PAUSED Status**

Khi nhận trạng thái PAUSED, UI nên thực hiện theo thứ tự:

1. **Hiển thị job paused interface** - Chuyển sang chế độ paused với styling đặc biệt
2. **Cập nhật status indicator** - Hiển thị "Tạm dừng" với icon pause và màu vàng
3. **Đóng băng job progress** - Giữ nguyên progress hiện tại với styling "frozen"
4. **Ẩn ETA display** - Không còn tính toán thời gian còn lại
5. **Cập nhật keyword display** - Hiển thị "Job đã tạm dừng"
6. **Hiển thị pause information** - Thông tin về thời gian tạm dừng và content đã thu thập
7. **Hiển thị action buttons tạm dừng** - Bật các nút "Tiếp tục" và "Hủy bỏ"
8. **Tắt content feed updates** - Dừng việc append content mới

### 3. Job Field Update Behaviors

Khi nhận message mới cho Job, các field được xử lý như sau:

| Field                     | Update Behavior | Mô tả                                               |
| ------------------------- | --------------- | --------------------------------------------------- |
| **`platform`**            | **OVERRIDE**    | Luôn cập nhật platform indicator                    |
| **`status`**              | **OVERRIDE**    | Luôn cập nhật trạng thái, kích hoạt UI state change |
| **`batch`**               | **OVERRIDE**    | Thay thế toàn bộ batch object (omit if empty)       |
| **`batch.keyword`**       | **OVERRIDE**    | Cập nhật keyword hiện tại đang được xử lý           |
| **`batch.content_list`**  | **APPEND**      | Thêm content mới vào đầu feed (newest first)        |
| **`batch.crawled_at`**    | **OVERRIDE**    | Cập nhật timestamp batch mới nhất                   |
| **`progress`**            | **OVERRIDE**    | Thay thế toàn bộ progress object (omit if empty)    |
| **`progress.current`**    | **OVERRIDE**    | Số batches đã hoàn thành (không cộng dồn)           |
| **`progress.total`**      | **OVERRIDE**    | Tổng số batches (có thể thay đổi)                   |
| **`progress.percentage`** | **OVERRIDE**    | Tính toán lại từ current/total                      |
| **`progress.eta`**        | **OVERRIDE**    | ETA mới nhất cho toàn bộ job (float64 minutes)      |
| **`progress.errors`**     | **OVERRIDE**    | Danh sách lỗi tổng hợp (không append)               |

**Lưu ý quan trọng:**

- **Omit empty behavior** - Nếu không có batch hoặc progress thì field đó không được gửi
- **Content list APPEND** - Mỗi batch mới append content vào feed hiện có
- **Duplicate check** - Kiểm tra content.id để tránh duplicate khi append
- **Progress OVERRIDE** - Tất cả progress metrics đều là giá trị tuyệt đối
- **Error aggregation** - Errors là tổng hợp từ tất cả batches, không phải chỉ batch hiện tại
- **Multiple messages per job** - Mỗi job sẽ có nhiều messages (mỗi batch hoàn thành = 1 message)
- **Real-time content feed** - UI nhận content tươi khi được crawl
