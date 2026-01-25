# Tasks: Add Phase-Based Progress

## Phase 1: Add Types (Day 1) ✅

### Task 1.1: Add Input Types ✅

- **File**: `internal/types/input.go`
- **Changes**: Thêm `PhaseProgressInput`, `ProjectPhaseInputMessage`, `ProjectPhasePayloadInput` structs
- **Validation**: Unit tests pass ✅
- **Dependencies**: None

### Task 1.2: Add Output Types ✅

- **File**: `internal/types/output.go`
- **Changes**: Thêm `PhaseProgress`, `ProjectPhaseNotificationMessage`, `ProjectPhasePayloadOutput` structs
- **Validation**: Unit tests pass ✅
- **Dependencies**: Task 1.1

### Task 1.3: Update Enums ✅

- **File**: `internal/types/enums.go`
- **Changes**: Thêm `INITIALIZING`, `PROCESSING`, `DONE` status constants, `PhaseBasedProjectStatus` type
- **Validation**: Compile pass ✅
- **Dependencies**: None

### Task 1.4: Write Input Type Tests ✅

- **File**: `internal/types/input_test.go`
- **Changes**: Thêm `TestPhaseProgressInput_Validate`, `TestProjectPhaseInputMessage_Validate`, `TestIsPhaseBasedMessage`
- **Validation**: `go test -v ./internal/types/...` ✅
- **Dependencies**: Task 1.1

---

## Phase 2: Update Transform Layer (Day 1-2) ✅

### Task 2.1: Add IsPhaseBasedMessage Helper ✅

- **File**: `internal/types/input.go`
- **Changes**: Thêm `IsPhaseBasedMessage(payload []byte) bool` function
- **Validation**: Unit tests pass ✅
- **Dependencies**: Task 1.1
- **Note**: Đã implemented trong Phase 1

### Task 2.2: Update Project Transformer ✅

- **File**: `internal/transform/project_transformer.go`
- **Changes**:
  - Thêm `TransformPhaseBased()` method
  - Thêm `transformPhaseToOutput()` helper
  - Thêm `transformPhaseProgress()` helper
  - Thêm `TransformAny()` để detect và route đúng format
- **Validation**: Unit tests pass ✅, backward compatible với legacy format
- **Dependencies**: Task 1.1, 1.2, 2.1

### Task 2.3: Update Validator ✅

- **File**: `internal/transform/validator.go`
- **Changes**:
  - Thêm `ValidateProjectPhaseInput()` method
  - Update `ValidateProjectInput()` để auto-detect và route đúng validator
- **Validation**: Unit tests pass ✅
- **Dependencies**: Task 1.1

### Task 2.4: Write Transformer Tests ✅

- **File**: `internal/transform/project_transformer_test.go`
- **Changes**: Thêm `TestProjectTransformer_TransformPhaseBased`, `TestProjectTransformer_TransformAny`:
  - Valid `project_progress` message ✅
  - Valid `project_completed` message ✅
  - INITIALIZING, FAILED status ✅
  - Invalid status/type/project_id ✅
  - Legacy format backward compatibility ✅
- **Validation**: `go test -v ./internal/transform/...` ✅
- **Dependencies**: Task 2.2

---

## Phase 3: Integration Testing (Day 2) ✅

### Task 3.1: Manual Redis Testing ✅

- **File**: `scripts/test-phase-progress.sh`
- **Tests**:
  - INITIALIZING status message
  - Crawl phase progress
  - Both phases (crawl + analyze) progress
  - COMPLETED (DONE) status message
  - FAILED status message
- **Validation**: Run `./scripts/test-phase-progress.sh all`
- **Dependencies**: Phase 1, Phase 2

### Task 3.2: Backward Compatibility Testing ✅

- **File**: `scripts/test-phase-progress.sh`
- **Tests**:
  - Legacy format message (without `type` field)
- **Validation**: Run `./scripts/test-phase-progress.sh legacy`
- **Dependencies**: Phase 1, Phase 2

---

## Phase 4: Documentation (Day 2-3) ✅

### Task 4.1: Write Client Migration Guide ✅

- **File**: `document/client-phase-progress-migration.md`
- **Content**:
  - TypeScript/JavaScript types cho new format ✅
  - `isPhaseBasedMessage()` helper function ✅
  - Example WebSocket message handler (React hook) ✅
  - Example React component with progress bars ✅
  - CSS styles ✅
  - Migration checklist ✅
  - Message examples ✅
  - FAQ section ✅
- **Dependencies**: Phase 2

---

## Phase 5: Deploy (Day 3)

### Task 5.1: Deploy to Staging

- **Validation**: Monitor logs, verify phase-based messages được xử lý đúng
- **Dependencies**: All previous phases

### Task 5.2: Production Deployment

- **Validation**: Monitor production logs và metrics
- **Dependencies**: Task 5.1

---

## Summary

| Phase | Tasks               | Estimated Duration | Parallelizable   |
| ----- | ------------------- | ------------------ | ---------------- |
| 1     | Types & Tests       | 1 day              | Tasks 1.1-1.3 ✓  |
| 2     | Transform Layer     | 1 day              | Tasks 2.1, 2.3 ✓ |
| 3     | Integration Testing | 0.5 day            | No               |
| 4     | Documentation       | 0.5 day            | ✓ with Phase 3   |
| 5     | Deploy              | 0.5 day            | No               |

**Total: ~3.5 days**
