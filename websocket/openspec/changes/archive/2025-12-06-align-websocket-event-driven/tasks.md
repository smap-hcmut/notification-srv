# Implementation Tasks

**Change ID**: `align-websocket-event-driven`  
**Status**: Draft

---

## Phase 1: Message Types & Payload (Priority: High)

### 1.1 Add Project Progress Message Types

- [x] Add `MessageTypeProjectProgress` constant to `message.go`
- [x] Add `MessageTypeProjectCompleted` constant to `message.go`
- Dependencies: None
- Verification: Unit test for type values

### 1.2 Add ProgressPayload Struct

- [x] Define `ProgressPayload` struct with JSON tags in `message.go`
- [x] Add validation method for payload fields
- Dependencies: 1.1
- Verification: Unit test for JSON serialization

### 1.3 Write Unit Tests for Messages

- [x] Create `message_test.go` with tests for new types
- [x] Test payload JSON marshal/unmarshal roundtrip
- Dependencies: 1.1, 1.2
- Verification: `go test -v ./internal/websocket/... -run TestProgress`

---

## Phase 2: Hub-Subscriber Integration (Priority: High)

### 2.1 Define RedisNotifier Interface

- [x] Add `RedisNotifier` interface to `hub.go`
- [x] Add optional `redisNotifier` field to Hub struct
- Dependencies: None
- Verification: Compile passes

### 2.2 Integrate Disconnect Callback

- [x] Update `unregisterConnection()` to call `OnUserDisconnected`
- [x] Add `SetRedisNotifier()` method for injection
- Dependencies: 2.1
- Verification: Unit test mock callback is invoked

### 2.3 Connect Subscriber to Hub

- [x] Wire subscriber as notifier in `cmd/server/main.go`
- Dependencies: 2.1, 2.2
- Verification: Integration test or manual verification

---

## Phase 3: Health Check Enhancement (Priority: Medium)

### 3.1 Add Subscriber Health Struct

- [x] Define `SubscriberHealth` struct in `health.go`
- [x] Add `Subscriber` field to `HealthResponse`
- Dependencies: Phase 2
- Verification: Compile passes

### 3.2 Implement Subscriber Health Method

- [x] Track `lastMessageAt` timestamp in subscriber
- [x] Track `isActive` atomic bool in subscriber
- [x] Implement `GetHealth()` method
- Dependencies: 3.1
- Verification: Unit test for health struct

### 3.3 Integrate Health in Handler

- [x] Update `healthHandler` to include subscriber status
- [x] Pass subscriber reference to server
- Dependencies: 3.1, 3.2
- Verification: `curl /health | jq '.subscriber'`

---

## Phase 4: Final Verification

### 4.1 Run Full Test Suite

- [x] Run `go test -v ./...`
- [x] Verify all tests pass
- Dependencies: All above
- Verification: Exit code 0

### 4.2 Manual E2E Test

- [x] Start service with `make run`
- [ ] Verify `/health` endpoint includes subscriber
- [ ] Connect WebSocket client and verify connection
- Dependencies: 4.1
- Verification: User approval

---

## Dependency Graph

```
Phase 1 (Message Types)
   1.1 → 1.2 → 1.3
        ↓
Phase 2 (Hub-Subscriber)
   2.1 → 2.2 → 2.3
        ↓
Phase 3 (Health Check)
   3.1 → 3.2 → 3.3
        ↓
Phase 4 (Verification)
   4.1 → 4.2
```
