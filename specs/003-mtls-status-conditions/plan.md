# Implementation Plan: mTLS Status Conditions on AgentRuntime

**Branch**: `003-mtls-status-conditions` | **Date**: 2026-06-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/003-mtls-status-conditions/spec.md`

## Summary

Add two Kubernetes status conditions (`ControlPlaneMTLS` and `DataPlaneMTLS`) to the AgentRuntime resource so platform engineers can see whether the controller-to-agent and agent-to-agent communication channels are secured with mTLS. Also add warning-level logging on plaintext fallback. All changes are in the AgentRuntime reconciler, using existing condition-setting patterns and the existing configuration resolution chain.

## Technical Context

**Language/Version**: Go 1.23+ (controller-runtime operator)
**Primary Dependencies**: controller-runtime, apimachinery (`meta.SetStatusCondition`), go-spiffe v2
**Storage**: Kubernetes API server (AgentRuntime status subresource)
**Testing**: Go test with envtest (controller-runtime integration testing), existing E2E framework
**Target Platform**: Kubernetes 1.28+
**Project Type**: Kubernetes operator (controller)
**Performance Goals**: N/A (status conditions set during existing reconcile, no additional I/O)
**Constraints**: Must follow Constitution Principle I (save/restore status across Patch calls)
**Scale/Scope**: 2 new condition types, ~150 lines of production code, ~200 lines of test code

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Reconciler Status Integrity | **PASS** | FR-011 explicitly requires save/restore pattern. The existing `fetchAndUpdateCard` → `persistCardFetchAnnotation` path already does a metadata Patch after in-memory status mutations. The new conditions must be set BEFORE `persistCardFetchAnnotation` to ensure they survive the Patch-induced refresh. |
| II. Spec-Anchored Testing | **PASS** | Acceptance scenarios say "inspect AgentRuntime status" which maps to envtest read-back. Tests will create objects in envtest and read them back. |
| III. Controller-Runtime Safety | **PASS** | Using existing `setCondition` helper which calls `meta.SetStatusCondition` on in-memory object. `Status().Update()` is called once at the end of reconcile. No additional Patch/Update calls introduced. |
| IV. CRD-First Design | **PASS** | Using standard `[]metav1.Condition` already on AgentRuntimeStatus. No CRD schema change needed. |
| V. Feature-Gated Rollout | **PASS** | `ControlPlaneMTLS` is inherently gated by `--enable-verified-fetch`. `DataPlaneMTLS` is status-only reporting (no behavior change), acceptable without a separate gate. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```text
specs/003-mtls-status-conditions/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (created by /speckit-tasks)
```

### Source Code (repository root)

```text
kagenti-operator/
├── api/v1alpha1/
│   └── agentruntime_types.go          # Condition type constants (no schema change)
├── internal/controller/
│   ├── agentruntime_controller.go     # Main changes: condition setting in fetchAndUpdateCard + reconcile
│   └── agentruntime_controller_test.go # New test cases for conditions
└── internal/agentcard/
    └── fetcher.go                     # Already returns FetchResult with AgentSpiffeID (no change)
```

**Structure Decision**: All changes land in the existing controller package. No new files, packages, or directories needed. The two conditions are set within the existing reconcile flow.

## Phase 0: Research

### R1: Status condition placement in the reconcile flow

**Decision**: Set `ControlPlaneMTLS` inside `fetchAndUpdateCard()` immediately after the fetch result is known. Set `DataPlaneMTLS` in the main `Reconcile()` method after `ComputeConfigHash()` resolves the config chain.

**Rationale**: `fetchAndUpdateCard()` already knows whether mTLS or plaintext was used (it checks `AuthenticatedFetcher` and `FetchResult`). The config chain is resolved in step 5 of `Reconcile()` which already has `configResult` with the merged `MTLSMode` and `AuthBridgeMode`.

**Alternatives considered**: Setting both conditions in a dedicated method. Rejected because it would require threading fetch result data through additional parameters when the information is already available at the right call sites.

### R2: Save/restore pattern compliance

**Decision**: No additional save/restore needed for the new conditions.

**Rationale**: The existing reconcile flow calls `setCondition()` (in-memory mutation) then `Status().Update()` at the end. The dangerous pattern is `Patch` or `Update` on the main resource between condition mutations and `Status().Update()`. The only such call in the path is `persistCardFetchAnnotation()` which does a metadata Patch. The existing code already handles this by calling it at the end of `fetchAndUpdateCard()`. The new `ControlPlaneMTLS` condition will be set BEFORE `persistCardFetchAnnotation()`, and `Status().Update()` happens after `fetchAndUpdateCard()` returns. The Patch in `persistCardFetchAnnotation` refreshes the object, but `Status().Update()` only writes the status subresource, so it will use whatever conditions are in memory at that point. However, since `persistCardFetchAnnotation` refreshes the object including status, we must save/restore `rt.Status` across that Patch call. This is the EXACT pattern documented in Constitution Principle I.

**Correction**: On closer inspection, `persistCardFetchAnnotation` does a `client.MergeFrom` Patch on the main resource, which replaces the local `rt` object with the API server response. This WILL wipe the in-memory conditions set earlier. The existing code works because `Status().Update()` is the last call and it overwrites the status subresource. But the conditions set in `fetchAndUpdateCard()` would be lost after the Patch. The fix: save `rt.Status` before `persistCardFetchAnnotation()` and restore it after. This pattern already exists conceptually in the codebase.

### R3: Condition type constants

**Decision**: Add two constants to the agentruntime controller: `ConditionTypeControlPlaneMTLS = "ControlPlaneMTLS"` and `ConditionTypeDataPlaneMTLS = "DataPlaneMTLS"`.

**Rationale**: Follows the existing pattern (`ConditionTypeTargetResolved`, `ConditionTypeReady`, `ConditionTypeCardSynced`, `ConditionTypeConfigResolved`). Constants live in the controller package, not the API package, because they're reconciler implementation details, not CRD schema.

### R4: Warning log format

**Decision**: Use structured logging: `logger.Info("Falling back to plaintext HTTP for card fetch", "agent", ref.Name, "namespace", rt.Namespace)` at V(0) level (equivalent to warning). The existing fallback path already has `logger.Info("TLS port not found, falling back to HTTP fetch")` and an event. We'll keep the event and add the structured log with agent context.

**Rationale**: The controller uses `logr` which doesn't have a `Warn` level. V(0) is the standard "always shown" level. The existing pattern uses `logger.Info` for operational messages and `logger.Error` for errors.

## Phase 1: Design & Contracts

### Data Model

No CRD schema changes. The existing `Conditions []metav1.Condition` field on `AgentRuntimeStatus` accommodates any number of condition types without schema modification.

**New condition types** (constants in controller package):

| Condition Type | Source | Semantics |
|---------------|--------|-----------|
| `ControlPlaneMTLS` | `fetchAndUpdateCard()` | Observed: did the last fetch use mTLS? |
| `DataPlaneMTLS` | `Reconcile()` step 5 | Configured: what mTLS mode is resolved for this agent? |

**ControlPlaneMTLS states:**

| Status | Reason | Message template | When |
|--------|--------|-----------------|------|
| `True` | `mTLS` | `"Peer SPIFFE ID: {spiffeID}"` | Authenticated fetch succeeded |
| `False` | `PlainHTTP` | `"SPIRE unavailable or TLS port not found; using plaintext HTTP"` | Fell back to unauthenticated fetch |
| `False` | `Disabled` | `"Verified fetch not enabled (--enable-verified-fetch=false)"` | `AuthenticatedFetcher` is nil |
| `False` | `FetchSkipped` | `"Card fetch skipped (no change detected)"` | Change key matched, no fetch performed |

**DataPlaneMTLS states:**

| Status | Reason | Message template | When |
|--------|--------|-----------------|------|
| `True` | `Strict` | `"mTLS strict mode configured"` | `resolved.MTLSMode == "strict"` |
| `True` | `Permissive` | `"mTLS permissive mode configured; accepts both TLS and plaintext"` | `resolved.MTLSMode == "permissive"` |
| `False` | `Disabled` | `"mTLS disabled"` | `resolved.MTLSMode == ""` or `"disabled"` |
| `False` | `NoSidecar` | `"No authbridge sidecar configured"` | `resolved.AuthBridgeMode == ""` |

### Integration Points

**fetchAndUpdateCard() changes:**

1. When `AuthenticatedFetcher != nil` and fetch succeeds: set `ControlPlaneMTLS` True/mTLS with SPIFFE ID
2. When `AuthenticatedFetcher != nil` but TLS port not found (fallback): set `ControlPlaneMTLS` False/PlainHTTP + warning log
3. When `AuthenticatedFetcher == nil`: set `ControlPlaneMTLS` False/Disabled
4. When fetch is skipped (change key match): set `ControlPlaneMTLS` False/FetchSkipped
5. Save/restore `rt.Status` across `persistCardFetchAnnotation()` Patch

**Reconcile() changes (after step 5, config hash):**

1. Check `configResult` resolved config for `AuthBridgeMode` and `MTLSMode`
2. Set `DataPlaneMTLS` based on resolved values

### No external contracts

This feature does not expose new external interfaces. The conditions are standard Kubernetes status conditions queryable via `kubectl` and the Kubernetes API.
