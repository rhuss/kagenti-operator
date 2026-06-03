# Data Model: mTLS Status Conditions

## No CRD Schema Changes

The existing `AgentRuntimeStatus.Conditions` field (`[]metav1.Condition`) accommodates new condition types without schema modification. No changes to `api/v1alpha1/agentruntime_types.go` struct definitions.

## New Condition Types

### ControlPlaneMTLS

Reports observed state of controller-to-agent communication security.

| Field | Value |
|-------|-------|
| Type | `ControlPlaneMTLS` |
| Status | `True` (mTLS used) or `False` (plaintext/disabled/skipped) |
| Reason | `mTLS`, `PlainHTTP`, `Disabled`, `FetchSkipped` |
| Message | Human-readable description with SPIFFE ID when available |
| ObservedGeneration | `rt.Generation` |
| LastTransitionTime | Auto-set by `meta.SetStatusCondition` |

**State transitions**:
- Startup without SPIRE: `False/Disabled` or `False/PlainHTTP`
- SPIRE becomes available: transitions to `True/mTLS` on next reconcile
- SPIRE goes down: transitions to `False/PlainHTTP` on next failed fetch
- No template change detected: `False/FetchSkipped` (condition preserves last fetch result)

### DataPlaneMTLS

Reports configured state of agent-to-agent communication security.

| Field | Value |
|-------|-------|
| Type | `DataPlaneMTLS` |
| Status | `True` (strict or permissive) or `False` (disabled/no sidecar) |
| Reason | `Strict`, `Permissive`, `Disabled`, `NoSidecar` |
| Message | Human-readable description of configured mode |
| ObservedGeneration | `rt.Generation` |
| LastTransitionTime | Auto-set by `meta.SetStatusCondition` |

**State transitions**:
- CR updates `mtlsMode`: condition updates on next reconcile
- Namespace ConfigMap overrides change: condition updates when config hash changes
- Sidecar removed: transitions to `False/NoSidecar`

## Existing Entities (unchanged)

- `AgentRuntimeStatus.Card.AttestedAgentSpiffeID`: continues to store the SPIFFE ID from verified fetch on the card status
- `AgentCard.Status.Conditions` (Verified): continues to track signature verification on AgentCard, not affected by this feature
