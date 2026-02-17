# Expected Output

---

## 1. Real-time Data Onboarding Progress

**Ai gửi:** Ingest Service (khi user upload file Excel, CSV, hoặc kết nối TikTok/YouTube)

**Service làm gì:**

- Nhận message từ Redis channel `project:{project_id}:user:{user_id}`
- Validate + transform message
- Push qua WebSocket tới browser của user đó

**User thấy gì trên UI:**

```
"Đang phân tích schema... 30%"
"Đang mapping dữ liệu... 60%"
"Import hoàn tất: 1,000 records, 2 lỗi"
```

---

## 2. Real-time Analytics Pipeline Progress

**Ai gửi:** Analytics Service (khi chạy sentiment analysis, aspect extraction, keyword extraction)

**Service làm gì:**

- Nhận message từ Redis channel `project:{project_id}:user:{user_id}`
- Validate + transform message
- Push qua WebSocket tới browser

**User thấy gì trên UI:**

```
"Đang phân tích sentiment... 300/1000 (30%) - Còn ~2 phút"
"Phase: ASPECT extraction... 70%"
"Phân tích hoàn tất: 980 thành công, 20 thất bại"
```

---

## 3. Crisis Alert (WebSocket + Discord)

**Ai gửi:** Analytics Service (khi phát hiện negative sentiment vượt ngưỡng)

**Service làm gì:**

- Nhận message từ Redis channel `alert:crisis:user:{user_id}`
- Validate + transform message
- **Đồng thời 2 kênh:**
  - Push qua WebSocket → UI hiện red alert banner
  - Post qua Discord webhook → Team nhận alert trên channel #smap-alerts

**User thấy gì trên UI:**

```
🚨 Crisis Alert: VF8 Monitor
   Negative sentiment 75% (ngưỡng 70%)
   Khía cạnh bị ảnh hưởng: PIN, GIÁ
   "Pin sụt nhanh quá", "Giá quá đắt"
   → Cần review feedback về PIN và GIÁ
```

**Team thấy gì trên Discord:**

```
🚨 Crisis Alert: VF8 Monitor
   Severity: HIGH
   Metric: Negative sentiment 75.0% (threshold: 70.0%)
   Affected: BATTERY, PRICE
   Action Required: Review negative feedback
```

---

## 4. Campaign Event Notification

**Ai gửi:** Knowledge Service (khi generate report xong, tạo artifact)

**Service làm gì:**

- Nhận message từ Redis channel `campaign:{campaign_id}:user:{user_id}`
- Validate + transform message
- Push qua WebSocket → UI hiện notification
- Post qua Discord → Team biết report đã sẵn sàng

**User thấy gì trên UI:**

```
📢 Báo cáo "So sánh Xe điện Q1" đã hoàn thành [Download]
```

---

## Tổng hợp: Service cung cấp gì?

| Chức năng                   | Input (Redis Pub/Sub)     | Output WebSocket | Output Discord       | Mục đích                              |
| --------------------------- | ------------------------- | ---------------- | -------------------- | ------------------------------------- |
| Data Onboarding Progress    | Ingest Service publish    | Push tới user    | Chỉ COMPLETED/FAILED | User biết tiến trình upload           |
| Analytics Pipeline Progress | Analytics Service publish | Push tới user    | Không                | User biết tiến trình phân tích        |
| Crisis Alert                | Analytics Service publish | Push tới user    | Luôn gửi             | Team phản ứng nhanh với khủng hoảng   |
| Campaign Event              | Knowledge Service publish | Push tới user    | Luôn gửi             | User biết report/artifact đã sẵn sàng |

**Ngoài 4 chức năng business, service còn cung cấp:**

| Chức năng infra       | Mô tả                                                   |
| --------------------- | ------------------------------------------------------- |
| JWT Authentication    | Xác thực user qua HttpOnly cookie hoặc Bearer token     |
| Connection Management | Hub quản lý max 10,000 concurrent WebSocket connections |
| Ping/Pong Keep-alive  | Giữ connection sống, detect disconnect                  |
| Health Checks         | `/health`, `/ready`, `/live` cho Kubernetes probes      |
| Graceful Shutdown     | Đóng connections + subscriber sạch khi restart          |
| CORS                  | Environment-aware (strict production, permissive dev)   |

---

**Nói ngắn gọn:** Service này là một **real-time notification hub** -- nhận event từ các microservice khác qua Redis, rồi push tới browser (WebSocket) và team (Discord). Service **không xử lý business logic**, chỉ **validate, transform, và route message** tới đúng người, đúng kênh.
