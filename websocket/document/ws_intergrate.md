# Hướng dẫn Tích hợp WebSocket cho Frontend

Tài liệu này chi tiết cách Frontend (FE) tích hợp với dịch vụ WebSocket (`smap-websocket`).

## 1. Chi tiết Kết nối

### Endpoint
- **URL**: `wss://smap-api.tantai.dev/ws` (Production)
- **Method**: `GET`

### Xác thực (Authentication)
Dịch vụ hỗ trợ 2 phương thức xác thực, yêu cầu JWT hợp lệ.

1.  **Cookie (Khuyến nghị)**:
    - Tên: `access_token`
    - Giá trị: `{jwt_token}`
    - Trình duyệt sẽ tự động gửi cookie này nếu cùng domain.

2.  **Query Parameter (Dự phòng)**:
    - URL: `/ws?token={jwt_token}`
    - Sử dụng phương thức này nếu không thể dùng cookie (ví dụ: khi dev khác domain).

### Hành vi Kết nối
- **Origins**:
    - **Production**: Danh sách whitelist nghiêm ngặt (`https://smap.tantai.dev`, v.v.).
    - **Dev/Staging**: Cho phép `localhost`, `127.0.0.1`, và các mạng nội bộ (`192.168.x.x`, v.v.).
- **Đa tab (Multi-tab)**: Hỗ trợ. Một user có thể mở nhiều tab/kết nối cùng lúc. Tin nhắn sẽ được gửi đến tất cả các kết nối đó.

---

## 2. Giao thức & Heartbeat

- **Hướng**: **Server-to-Client** (một chiều từ Server xuống Client).
- **Client Messages**: Server sẽ **bỏ qua** mọi tin nhắn text/binary gửi từ Client lên.
- **Heartbeat (Ping/Pong)**:
    - Server gửi **Ping** frame mỗi `54s` (xấp xỉ).
    - Client (trình duyệt/thư viện websocket) **BẮT BUỘC** phản hồi bằng **Pong** frame.
    - *Lưu ý*: Hầu hết các thư viện WebSocket chuẩn (như `WebSocket` API của trình duyệt) đều tự động xử lý Ping/Pong. Bạn thường không cần code tay phần này.
- **Ngắt kết nối**:
    - Nếu Server không nhận được Pong trong vòng `60s`, kết nối sẽ bị đóng.

---

## 3. Định dạng Tin nhắn

Tất cả tin nhắn từ Server gửi xuống Client đều tuân theo cấu trúc bao đóng (envelope) chuẩn sau:

```typescript
interface WSMessage<T = any> {
  type: MessageType;       // String enum
  payload: T;              // Dữ liệu thực tế
  timestamp: string;       // Chuỗi ISO 8601 (vd: "2025-12-09T10:00:00Z")
}
```

### Các loại Tin nhắn (`MessageType`)

| Type | Mô tả | Cấu trúc Payload | Nguồn dữ liệu (Backend) |
| :--- | :--- | :--- | :--- |
| `project_progress` | Cập nhật tiến độ của dự án crawling | `ProgressPayload` | `webhook.ProgressRequest` |
| `project_completed` | Thông báo dự án đã hoàn thành | `ProgressPayload` | `webhook.ProgressRequest` |
| `dryrun_result` | Kết quả của quá trình chạy thử (dry-run) | `DryRunPayload` | `project.CallbackRequest` |


---

## 4. Chi tiết Payload (Data Structures)

Dưới đây là định nghĩa chi tiết các struct payload dựa trên source code `collector/internal`.

### A. Progress Payload (`project_progress` & `project_completed`)
Gửi khi trạng thái dự án thay đổi hoặc có cập nhật tiến độ.

**Source**: `collector/internal/webhook/types.go`

```typescript
interface ProgressPayload {
  project_id: string;
  status: 'INITIALIZING' | 'CRAWLING' | 'PROCESSING' | 'DONE' | 'FAILED';
  total: number;    // Tổng số item cần xử lý
  done: number;     // Số item đã xử lý thành công
  errors: number;   // Số lượng item bị lỗi
  

  // Backend có gửi thêm field
  progress_percent?: number; 
}
```

**Ví dụ JSON**:
```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_123abc",
    "status": "CRAWLING",
    "total": 500,
    "done": 125,
    "errors": 2,
    "progress_percent": 25.0
  },
  "timestamp": "2025-12-09T10:15:00Z"
}
```

### B. Dry Run Result Payload (`dryrun_result`)
Gửi khi một quá trình "dry run" (cào thử) hoàn tất. Cấu trúc này rất chi tiết, chứa kết quả cào được.

**Source**: `collector/pkg/project/type.go` (`CallbackRequest`)

```typescript
interface DryRunPayload {
  job_id: string;
  status: 'success' | 'failed';
  platform: string; // e.g., "facebook", "tiktok", "youtube"
  content?: Content[]; // Mảng các nội dung cào được, nằm trực tiếp trong payload
  errors?: CrawlError[]; // Mảng các lỗi, nằm trực tiếp trong payload
}

interface CrawlError {
  code: string;
  message: string;
  keyword?: string;
}

// Cấu trúc Content chi tiết
interface Content {
  meta: ContentMeta;
  content: ContentData;
  interaction: ContentInteraction;
  author: ContentAuthor;
  comments?: Comment[];
}

interface ContentMeta {
  id: string;             // ID duy nhất của post trên platform
  platform: string;
  job_id: string;
  crawled_at: string;     // ISO 8601
  published_at: string;   // ISO 8601
  permalink: string;      // Link gốc tới bài viết
  keyword_source: string;
  lang: string;
  region: string;
  pipeline_version: string;
  fetch_status: string;
  fetch_error?: string;
}

interface ContentData {
  text: string;
  duration?: number;
  hashtags?: string[];
  sound_name?: string;
  category?: string;
  title?: string;         // YouTube only
  transcription?: string;
  media?: {
    type: string;         // "video", "image", etc.
    video_path?: string;
    audio_path?: string;
    downloaded_at?: string;
  };
}

interface ContentInteraction {
  views: number;
  likes: number;
  comments_count: number;
  shares: number;
  saves?: number;
  engagement_rate?: number;
  updated_at: string;
}

interface ContentAuthor {
  id: string;
  name: string;
  username: string;
  followers: number;
  following: number;
  likes: number;
  videos: number;
  is_verified: boolean;
  bio?: string;
  avatar_url?: string;
  profile_url: string;
  country?: string;           // YouTube only
  total_view_count?: number;  // YouTube only
}

interface Comment {
  id: string;
  parent_id?: string;
  post_id: string;
  user: {
    id?: string;
    name: string;
    avatar_url?: string;
  };
  text: string;
  likes: number;
  replies_count: number;
  published_at: string;
  is_author: boolean;
  media?: string;
  is_favorited: boolean; // YouTube only
}
```

**Ví dụ JSON (Rút gọn)**:
```json
{
  "type": "dryrun_result",
  "payload": {
    "job_id": "job_dry_run_001",
    "status": "success",
    "platform": "facebook",
    "content": [
      {
        "meta": {
          "id": "123456789",
          "platform": "facebook",
          "job_id": "job_dry_run_001",
          "crawled_at": "2025-12-09T10:20:00Z",
          "published_at": "2025-12-08T15:00:00Z",
          "permalink": "https://facebook.com/posts/123456789",
          "keyword_source": "covid19",
          "lang": "vi",
          "region": "VN",
          "pipeline_version": "v1.0",
          "fetch_status": "success"
        },
        "content": {
          "text": "Nội dung bài viết mẫu...",
          "hashtags": ["news", "update"]
        },
        "interaction": {
          "views": 1000,
          "likes": 50,
          "comments_count": 10,
          "shares": 5,
          "updated_at": "2025-12-09T10:20:00Z"
        },
        "author": {
          "id": "author_001",
          "name": "Tin Tức 24h",
          "username": "tintuc24h",
          "followers": 50000,
          "following": 10,
          "likes": 100000,
          "videos": 0,
          "is_verified": true,
          "profile_url": "https://facebook.com/tintuc24h"
        }
      }
    ]
  },
  "timestamp": "2025-12-09T10:20:00Z"
}
```

---

## 6. Xử lý Lỗi (Error Handling)

### Lỗi Kết nối (Handshake Errors)
Khi thiết lập kết nối (`GET /ws`), Server có thể trả về các HTTP Error Code sau:

| Code | Lý do | Hành động khuyến nghị |
| :--- | :--- | :--- |
| `401 Unauthorized` | 1. Không có token.<br>2. Token không hợp lệ/hết hạn. | Redirect user về trang Login để lấy token mới. |
| `1006 Abnormal Closure` | Kết nối bị ngắt đột ngột (network, server reset). | Thử reconnect với chiến thuật **Exponential Backoff**. |

### Lỗi Vận hành (Runtime Errors)

- **Max Connections Reached**: Nếu User mở quá nhiều tab (vượt giới hạn config của server), các kết nối mới có thể bị từ chối hoặc đóng ngay lập tức. Client nên handle sự kiện `onclose` và thử lại sau.
- **Buffer Full**: Nếu Client xử lý chậm và để hàng đợi tin nhắn trên Server bị đầy, Server sẽ drop tin nhắn đó để bảo vệ hệ thống. Đây là cơ chế fail-safe, Client sẽ không nhận được thông báo lỗi cụ thể qua WS frame mà chỉ thấy mất tin nhắn.
- **Ping/Pong Timeout**: Nếu Client mất mạng hoặc không phản hồi Ping trong `60s`, Server sẽ chủ động đóng kết nối. Client cần listen sự kiện `onclose` để reconnect.


## 5. Ví dụ Tích hợp (React)

```typescript
import { useEffect } from 'react';

const WS_URL = 'wss://smap-api.tantai.dev/ws'; // Nên dùng biến môi trường

export const useWebSocket = (token: string) => {
  useEffect(() => {
    if (!token) return;

    // Kết nối với token qua query param (hoặc dựa vào cookie nếu có)
    const ws = new WebSocket(`${WS_URL}?token=${token}`);

    ws.onopen = () => {
      console.log('Đã kết nối WebSocket');
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
      } catch (err) {
        console.error('Lỗi parse tin nhắn WS', err);
      }
    };

    ws.onclose = () => {
      console.log('Đã ngắt kết nối');
      // Thêm logic reconnect tại đây nếu cần
    };

    return () => {
      ws.close();
    };
  }, [token]);

  const handleMessage = (msg: any) => {
    switch (msg.type) {
      case 'project_progress':
        console.log('Tiến độ:', msg.payload.progress_percent + '%');
        // Update state UI
        break;
      case 'dryrun_result':
        console.log('Kết quả Dry Run:', msg.payload);
        // Hiển thị kết quả mẫu cho user xem
        break;
      case 'notification':
        // Hiển thị toaster
        break;
      default:
        console.warn('Loại tin nhắn không xác định:', msg.type);
    }
  };
};
```
