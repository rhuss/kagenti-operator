# Tasks: mTLS Status Conditions on AgentRuntime

**Input**: Design documents from `specs/003-mtls-status-conditions/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md

**Tests**: Included. This is a Kubernetes operator feature where envtest-based tests are essential for correctness per Constitution Principle II.

**Organization**: Tasks grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Add condition type constants and ensure the save/restore pattern is in place

- [X] T001 Add condition type constants `ConditionTypeControlPlaneMTLS` and `ConditionTypeDataPlaneMTLS` in `kagenti-operator/internal/controller/agentruntime_controller.go`
- [X] T002 Add save/restore for `rt.Status` around `persistCardFetchAnnotation()` call in `fetchAndUpdateCard()` in `kagenti-operator/internal/controller/agentruntime_controller.go`

---

## Phase 2: User Story 1 - Control-Plane mTLS Visibility (Priority: P1)

**Goal**: Platform engineers can see whether the controller used mTLS or plaintext for each agent card fetch by inspecting a single condition on AgentRuntime.

**Independent Test**: Deploy an agent with verified fetch enabled, inspect AgentRuntime status for `ControlPlaneMTLS` condition showing reason `mTLS` with SPIFFE ID or reason `PlainHTTP`/`Disabled`.

### Tests for User Story 1

- [X] T003 [P] [US1] Add envtest case: verified fetch succeeds with mTLS, assert `ControlPlaneMTLS` condition True/mTLS with SPIFFE ID in message, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`
- [X] T004 [P] [US1] Add envtest case: verified fetch falls back to plaintext (no TLS port), assert `ControlPlaneMTLS` condition False/PlainHTTP, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`
- [X] T005 [P] [US1] Add envtest case: verified fetch disabled (`AuthenticatedFetcher` nil), assert `ControlPlaneMTLS` condition False/Disabled, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`
- [X] T006 [P] [US1] Add envtest case: card fetch skipped (change key match), assert `ControlPlaneMTLS` condition False/FetchSkipped, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`

### Implementation for User Story 1

- [X] T007 [US1] Set `ControlPlaneMTLS` True/mTLS with SPIFFE ID when `AuthenticatedFetcher` succeeds in `fetchAndUpdateCard()` in `kagenti-operator/internal/controller/agentruntime_controller.go`
- [X] T008 [US1] Set `ControlPlaneMTLS` False/PlainHTTP when falling back to HTTP fetch (TLS port not found) in `fetchAndUpdateCard()` in `kagenti-operator/internal/controller/agentruntime_controller.go`
- [X] T009 [US1] Set `ControlPlaneMTLS` False/Disabled when `AuthenticatedFetcher` is nil in `fetchAndUpdateCard()` in `kagenti-operator/internal/controller/agentruntime_controller.go`
- [X] T010 [US1] Set `ControlPlaneMTLS` False/FetchSkipped when fetch is skipped due to change key match in `fetchAndUpdateCard()` in `kagenti-operator/internal/controller/agentruntime_controller.go`

**Checkpoint**: `ControlPlaneMTLS` condition visible on AgentRuntime after card fetch. All 4 states tested.

---

## Phase 3: User Story 2 - Data-Plane mTLS Visibility (Priority: P2)

**Goal**: Platform engineers can see the resolved mTLS mode for each agent's data-plane communication by inspecting a single condition on AgentRuntime.

**Independent Test**: Deploy an agent with `mtlsMode: strict`, inspect AgentRuntime status for `DataPlaneMTLS` condition showing reason `Strict`.

### Tests for User Story 2

- [X] T011 [P] [US2] Add envtest case: AgentRuntime with `mtlsMode: strict` and `authBridgeMode: proxy-sidecar`, assert `DataPlaneMTLS` condition True/Strict, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`
- [X] T012 [P] [US2] Add envtest case: AgentRuntime with `mtlsMode: permissive`, assert `DataPlaneMTLS` condition True/Permissive, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`
- [X] T013 [P] [US2] Add envtest case: AgentRuntime with `mtlsMode: disabled` or empty, assert `DataPlaneMTLS` condition False/Disabled, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`
- [X] T014 [P] [US2] Add envtest case: AgentRuntime with no `authBridgeMode`, assert `DataPlaneMTLS` condition False/NoSidecar, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`

### Implementation for User Story 2

- [X] T015 [US2] Add `DataPlaneMTLS` condition logic after `ComputeConfigHash()` in `Reconcile()`, reading resolved `AuthBridgeMode` and `MTLSMode` from `configResult`, in `kagenti-operator/internal/controller/agentruntime_controller.go`
- [X] T016 [US2] Expose resolved `AuthBridgeMode` and `MTLSMode` from `ComputeConfigHash()` return value (add fields to `ConfigResult` struct if not already present) in `kagenti-operator/internal/controller/agentruntime_config.go`

**Checkpoint**: `DataPlaneMTLS` condition visible on AgentRuntime reflecting resolved config chain. All 4 states tested.

---

## Phase 4: User Story 3 - Warning Logging on Plaintext Fallback (Priority: P3)

**Goal**: Warning-level log emitted on each plaintext fallback with agent name and namespace for troubleshooting.

**Independent Test**: Disable SPIRE, trigger reconcile, check controller logs for warning-level message with agent context.

### Tests for User Story 3

- [X] T017 [US3] Add test case: verify warning log emitted on plaintext fallback containing agent name and namespace, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`

### Implementation for User Story 3

- [X] T018 [US3] Add structured warning log at V(0) level in the HTTP fallback path of `fetchCard()`, including agent name (`ref.Name`) and namespace (`rt.Namespace`), in `kagenti-operator/internal/controller/agentruntime_controller.go`

**Checkpoint**: Plaintext fallback produces a warning log with agent context. Existing Event emission preserved.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Verify integration across all conditions

- [X] T019 Add envtest case: SPIRE transitions from unavailable to available, verify `ControlPlaneMTLS` transitions from False/PlainHTTP to True/mTLS across reconcile cycles, in `kagenti-operator/internal/controller/agentruntime_controller_test.go`
- [X] T020 Verify save/restore pattern: add envtest case confirming conditions survive the `persistCardFetchAnnotation()` Patch (read back from API server after reconcile), in `kagenti-operator/internal/controller/agentruntime_controller_test.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies, start immediately
- **User Story 1 (Phase 2)**: Depends on Phase 1 (constants and save/restore pattern)
- **User Story 2 (Phase 3)**: Depends on Phase 1 only (independent of US1)
- **User Story 3 (Phase 4)**: Depends on Phase 1 only (independent of US1 and US2)
- **Polish (Phase 5)**: Depends on US1 and US2 completion

### User Story Dependencies

- **US1 (P1)**: Constants + save/restore from Phase 1. No dependency on other stories.
- **US2 (P2)**: Constants from Phase 1. May need `ConfigResult` struct extension (T016). No dependency on US1.
- **US3 (P3)**: No dependency on US1 or US2. Can be implemented in parallel.

### Within Each User Story

- Tests written first (envtest cases), implementation follows
- All test tasks within a story are parallelizable [P]

### Parallel Opportunities

- T003, T004, T005, T006 can all run in parallel (separate test cases)
- T011, T012, T013, T014 can all run in parallel (separate test cases)
- US1, US2, US3 can be implemented in parallel after Phase 1

---

## Parallel Example: User Story 1

```bash
# Launch all tests for US1 together:
Task: "T003 envtest: mTLS fetch succeeds"
Task: "T004 envtest: plaintext fallback"
Task: "T005 envtest: verified fetch disabled"
Task: "T006 envtest: fetch skipped"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001, T002)
2. Complete Phase 2: User Story 1 (T003-T010)
3. **STOP and VALIDATE**: `kubectl get agentruntime -o yaml` shows `ControlPlaneMTLS` condition
4. Deploy/demo if ready

### Incremental Delivery

1. Setup → US1 → Validate (MVP: control-plane visibility)
2. Add US2 → Validate (data-plane visibility)
3. Add US3 → Validate (warning logging)
4. Polish → Validate (transition + save/restore edge cases)

---

## Notes

- All test tasks use envtest with API server read-back per Constitution Principle II
- T002 (save/restore) is critical infrastructure; T020 validates it end-to-end
- All tasks target a single file (`agentruntime_controller.go` or `agentruntime_config.go`) to minimize merge conflicts
- Commit after each task or logical group
