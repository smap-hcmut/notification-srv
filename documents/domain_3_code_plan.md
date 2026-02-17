# Cleanup & Migration Code Plan

**Ref:** `documents/master-proposal.md`, `documents/domain_convention/convention.md`
**Status:** DRAFT
**Domain:** `internal/websocket` (Tests), `internal/transform` (Deletion)

This document details the final cleanup and migration steps to fully satisfy the master proposal.

---

## 1. Deletion of Legacy Domain

The `internal/transform` domain is now obsolete as its logic has been integrated into `internal/websocket`.

- **Action**: Delete `internal/transform/` directory recursively.
- **Verification**: Ensure no imports remain in the codebase (checked `main.go`, need to check others if any).

## 2. Test Migration

The existing tests in `internal/websocket/` are broken because they reference old types and function signatures (e.g., `NewHub`, `NewHandlerWithOptions`, `MessageTypeJobCompleted`).

### 2.1 `internal/websocket/integration_test.go`

- **Status**: BROKEN.
- **Plan**: Rewrite to test the new `websocket` domain.
  - Use `websocket.New` (UseCase) instead of `usecase.NewHub`.
  - Use `http.New` (Handler) instead of `NewHandlerWithOptions`.
  - Test new message types (`DATA_ONBOARDING`, `CRISIS_ALERT`) instead of `PROJECT`/`JOB`.
  - Mock `alert.UseCase` dependency.

### 2.2 `internal/websocket/compatibility_test.go`

- **Status**: Likely BROKEN.
- **Plan**: Assess if backward compatibility is still needed. The master proposal implies a "Refactor", but also "Delete legacy types".
  - **Decision**: Since we deleted legacy types (`MessageTypeJob...`), backward compatibility with strict legacy types is impossible.
  - **Action**: Remove this file or adapt it to test that *new* clients work as expected. Given the major version change (refactor), we will REMOVE it or Replace with `functional_test.go`.

### 2.3 `internal/websocket/mock_rate_limiter_test.go` & `types_test.go`

- **Action**: Update to use new types or delete if irrelevant.

---

## 3. Implementation Checklist

- [ ] **Delete Legacy**:
  - [ ] Delete `internal/transform/`

- [ ] **Refactor Tests**:
  - [ ] Create `internal/websocket/tests/integration_test.go` (New Suite)
  - [ ] Create `internal/websocket/tests/mocks.go` (Mock AlertUC, Mock Repo)
  - [ ] Delete old test files in `internal/websocket/` (`integration_test.go`, `compatibility_test.go`).

- [ ] **Final Verification**:
  - [ ] Run `go test ./internal/websocket/...`
  - [ ] Run `go build ./...`

---
