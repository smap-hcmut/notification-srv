# Add Phase-Based Progress

## Summary

Hỗ trợ format message mới với phase-based progress từ Project Service. Thay đổi này cho phép hiển thị tiến độ chi tiết theo từng phase (crawl, analyze) thay vì chỉ một progress bar duy nhất.

**Related Documentation:**

- [ws-service-migration-phase-progress.md](file:///Users/tantai/Workspaces/tools/secondary/websocket/document/ws-service-migration-phase-progress.md)
- [phase-based-progress-implementation-guide.md](file:///Users/tantai/Workspaces/tools/secondary/websocket/document/phase-based-progress-implementation-guide.md)

## Why

Project Service đã chuyển sang sử dụng phase-based progress thay vì single progress bar. Thay đổi này cần thiết vì:

1. **Chi tiết hóa tiến độ**: Hiển thị riêng biệt progress cho crawl phase và analyze phase
2. **UX tốt hơn**: Frontend có thể hiển thị 2 progress bars độc lập, giúp user hiểu rõ project đang ở phase nào
3. **Error tracking**: Mỗi phase có riêng error count, giúp debug và monitoring dễ hơn
4. **Backward compatible**: Legacy format vẫn được hỗ trợ, không ảnh hưởng clients cũ

## What Changes

### Message Format

- **Thêm** `PhaseProgress` struct để đại diện tiến độ cho từng phase
- **Thêm** `ProjectPhaseInputMessage` và `ProjectPhaseNotificationMessage` types
- **Cập nhật** transformer để hỗ trợ cả legacy và phase-based format
- **Backward compatible**: Messages cũ vẫn hoạt động

### New Phase-Based Payload Structure

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

### Status Changes

- **Thêm** `INITIALIZING`, `PROCESSING` statuses mới
- **Giữ nguyên** `DONE`, `FAILED` cho completion states
- **Legacy** `CRAWLING`, `COMPLETED`, `PAUSED` vẫn được hỗ trợ

## User Review Required

> [!IMPORTANT] > **Backward Compatibility**: Thay đổi này backward compatible. Client cũ có thể ignore các fields mới (`crawl`, `analyze`, `overall_progress_percent`).

> [!NOTE] > **Channel Pattern**: Channel pattern mới (`project:{project_id}:{user_id}`) đã được hỗ trợ sẵn trong codebase hiện tại.

## Spec Deltas

| Capability                                             | Changes                                     |
| ------------------------------------------------------ | ------------------------------------------- |
| [websocket-messages](specs/websocket-messages/spec.md) | Thêm Phase-Based Payload requirement        |
| [message-transform](specs/message-transform/spec.md)   | Thêm Phase-Based Transformation requirement |

## Verification Plan

### Automated Tests

1. Unit tests cho `PhaseProgressInput.Validate()` và `ProjectPhaseInputMessage.Validate()`
2. Unit tests cho `ProjectTransformer.transformPhaseBasedMessage()`
3. Run: `go test ./internal/types/... ./internal/transform/...`

### Integration Testing

1. Publish test message qua Redis CLI:
   ```bash
   redis-cli PUBLISH "project:test_proj:test_user" '{"type": "project_progress", "payload": {"project_id": "test_proj", "status": "PROCESSING", "crawl": {"total": 100, "done": 80, "errors": 2, "progress_percent": 82.0}, "analyze": {"total": 78, "done": 45, "errors": 1, "progress_percent": 59.0}, "overall_progress_percent": 70.5}}'
   ```
2. Verify WebSocket client nhận đúng message format
