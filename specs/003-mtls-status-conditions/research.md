# Research: mTLS Status Conditions

## R1: Status condition placement in the reconcile flow

**Decision**: Set `ControlPlaneMTLS` inside `fetchAndUpdateCard()`. Set `DataPlaneMTLS` in `Reconcile()` after `ComputeConfigHash()`.

**Rationale**: Information is available at these exact call sites without additional parameter threading. `fetchAndUpdateCard()` already distinguishes mTLS vs plaintext (checks `AuthenticatedFetcher` and `FetchResult`). `Reconcile()` step 5 already resolves the merged config including `MTLSMode` and `AuthBridgeMode`.

**Alternatives considered**: Dedicated method collecting both conditions. Rejected because it would require passing fetch result data and resolved config through additional parameters.

## R2: Save/restore pattern compliance (Constitution Principle I)

**Decision**: Save `rt.Status` before `persistCardFetchAnnotation()` and restore after.

**Rationale**: `persistCardFetchAnnotation()` does a `client.MergeFrom` Patch on the main resource, which replaces the local `rt` object with the API server response. Any in-memory status mutations (including the new conditions) would be lost. The fix is the standard save/restore pattern from Constitution Principle I.

**Code pattern**:
```go
savedStatus := rt.Status.DeepCopy()
r.persistCardFetchAnnotation(ctx, rt, changeKey)
rt.Status = *savedStatus
```

## R3: Condition type constants

**Decision**: Add constants to the controller package following existing naming convention.

**Rationale**: Existing constants (`ConditionTypeTargetResolved`, `ConditionTypeReady`, `ConditionTypeCardSynced`, `ConditionTypeConfigResolved`) live in the controller package.

## R4: Warning log format

**Decision**: Structured log at V(0) level with agent name and namespace context fields.

**Rationale**: `logr` has no `Warn` level. V(0) is "always shown". Matches existing pattern in the controller. The existing fallback already emits a Kubernetes Event; the log adds structured context for log aggregation.
