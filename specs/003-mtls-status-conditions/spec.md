# Feature Specification: mTLS Status Conditions on AgentRuntime

**Feature Branch**: `003-mtls-status-conditions`
**Created**: 2026-06-02
**Status**: Draft
**Input**: Brainstorm `brainstorm/05-mtls-status-conditions.md`, Jira [RHAIENG-4928](https://redhat.atlassian.net/browse/RHAIENG-4928)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Control-Plane mTLS Visibility (Priority: P1)

As a platform engineer operating agents on a cluster with SPIRE deployed, I need to know whether the operator controller is communicating securely with each agent, so that I can verify the control-plane security posture and troubleshoot connectivity issues.

Today, the controller silently falls back to plaintext HTTP when SPIRE is unavailable. There is no status signal indicating which mode was used. This makes it impossible to audit control-plane security without reading controller logs.

**Why this priority**: This is the core gap identified in RHAIENG-4928. Without this condition, operators cannot verify that the control plane is actually using mTLS even when SPIRE is deployed and configured.

**Independent Test**: Can be fully tested by deploying an agent workload with verified fetch enabled, then inspecting the AgentRuntime status for the `ControlPlaneMTLS` condition. Delivers immediate visibility into control-plane security posture.

**Acceptance Scenarios**:

1. **Given** a cluster with SPIRE deployed and verified fetch enabled on the controller, **When** the controller successfully fetches an agent card over mTLS, **Then** the AgentRuntime status shows `ControlPlaneMTLS` condition with status `True`, reason `mTLS`, and a message containing the peer's SPIFFE ID.

2. **Given** a cluster without SPIRE or with SPIRE temporarily unavailable, **When** the controller fetches an agent card, **Then** the AgentRuntime status shows `ControlPlaneMTLS` condition with status `False`, reason `PlainHTTP`, and a message indicating the fallback.

3. **Given** a cluster where verified fetch is not enabled on the controller, **When** the controller fetches an agent card, **Then** the AgentRuntime status shows `ControlPlaneMTLS` condition with status `False`, reason `Disabled`, and a message indicating verified fetch is not enabled.

4. **Given** SPIRE was unavailable and the condition showed `PlainHTTP`, **When** SPIRE becomes available and the controller reconciles, **Then** the condition transitions to status `True`, reason `mTLS`.

---

### User Story 2 - Data-Plane mTLS Visibility (Priority: P2)

As a platform engineer, I need to know the configured mTLS mode for each agent's data-plane (agent-to-agent) communication, so that I can verify that agents are configured for mutual authentication as intended.

The `mtlsMode` field exists on the AgentRuntime spec, but there is no corresponding status condition reflecting the resolved configuration (which may come from cluster defaults, namespace ConfigMap, or the CR itself). Operators must trace through the configuration chain to determine the effective mode.

**Why this priority**: Provides the second security dimension. While less urgent than the control-plane condition (since the spec field already exists), having the resolved effective mode in status makes auditing practical across many agents.

**Independent Test**: Can be fully tested by deploying an agent with a specific `mtlsMode` and verifying the `DataPlaneMTLS` condition reflects the resolved mode, including namespace-level overrides.

**Acceptance Scenarios**:

1. **Given** an AgentRuntime with `mtlsMode: strict` and SPIRE enabled, **When** the sidecar is injected and the runtime reconciles, **Then** the status shows `DataPlaneMTLS` condition with status `True`, reason `Strict`.

2. **Given** an AgentRuntime with `mtlsMode: permissive`, **When** the sidecar is injected and the runtime reconciles, **Then** the status shows `DataPlaneMTLS` condition with status `True`, reason `Permissive`.

3. **Given** an AgentRuntime with `mtlsMode: disabled` or mTLS not configured, **When** the runtime reconciles, **Then** the status shows `DataPlaneMTLS` condition with status `False`, reason `Disabled`.

4. **Given** an AgentRuntime without sidecar injection (no authBridgeMode set), **When** the runtime reconciles, **Then** the status shows `DataPlaneMTLS` condition with status `False`, reason `NoSidecar`.

5. **Given** a namespace ConfigMap overrides `mtlsMode` to `strict` but the AgentRuntime CR specifies `permissive`, **When** the runtime reconciles, **Then** the condition reflects the resolved mode from the configuration chain (CR override wins).

---

### User Story 3 - Warning Logging on Plaintext Fallback (Priority: P3)

As a platform engineer reviewing controller logs, I need a warning log entry emitted each time the controller falls back to plaintext HTTP for an agent card fetch, so that I can detect and troubleshoot SPIRE connectivity issues without relying solely on status conditions.

**Why this priority**: Supplements the status conditions with real-time log signals. Lower priority because status conditions provide the persistent, queryable state. Logs add the time-series dimension for troubleshooting.

**Independent Test**: Can be tested by disabling SPIRE, triggering a reconcile, and checking controller logs for warning-level messages mentioning the agent name and namespace.

**Acceptance Scenarios**:

1. **Given** verified fetch is enabled but SPIRE is unavailable, **When** the controller fetches an agent card using plaintext HTTP, **Then** a warning-level log is emitted containing the agent name, namespace, and a message indicating plaintext fallback.

2. **Given** verified fetch is enabled and SPIRE is available, **When** the controller fetches an agent card over mTLS, **Then** no warning log is emitted for that fetch.

---

### Edge Cases

- What happens when SPIRE is available during controller startup but becomes unavailable mid-operation? The `ControlPlaneMTLS` condition should transition from `mTLS` to `PlainHTTP` on the next reconcile that encounters the failure, without requiring controller restart.
- What happens when an AgentRuntime has `mtlsMode: strict` but SPIRE is not deployed on the cluster? The `DataPlaneMTLS` condition should reflect the configured mode (`Strict`) but the sidecar will fail to start. This is a separate operational concern (sidecar health), not the status condition's responsibility.
- What happens during SPIRE agent pod restarts (brief unavailability)? The controller should tolerate short-lived SPIRE outages without flapping the condition. If a fetch succeeds on retry, the condition should remain `mTLS`. Only sustained failures should transition to `PlainHTTP`.
- What happens when multiple agent card fetches happen in rapid succession with mixed results (some mTLS, some plaintext)? The condition should reflect the outcome of the most recent fetch.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The operator MUST set a `ControlPlaneMTLS` status condition on AgentRuntime after each agent card fetch, reflecting whether the fetch used mTLS or plaintext HTTP
- **FR-002**: The `ControlPlaneMTLS` condition MUST include the peer's SPIFFE ID in the message when mTLS is used
- **FR-003**: The `ControlPlaneMTLS` condition MUST use reason `Disabled` when verified fetch is not enabled on the controller
- **FR-004**: The operator MUST set a `DataPlaneMTLS` status condition on AgentRuntime reflecting the resolved mTLS mode for the agent's data-plane communication
- **FR-005**: The `DataPlaneMTLS` condition MUST reflect the effective mode after resolving the configuration chain (cluster defaults, namespace ConfigMap, CR override)
- **FR-006**: The `DataPlaneMTLS` condition MUST use reason `NoSidecar` when no authbridge sidecar is configured for the agent
- **FR-007**: The controller MUST emit a warning-level log on each plaintext HTTP fallback, including the agent name and namespace
- **FR-008**: Status conditions MUST follow the standard Kubernetes condition convention (type, status, reason, message, lastTransitionTime)
- **FR-011**: Condition mutations MUST be resilient to API server round-trips during the reconcile cycle. If any `Patch` or `Update` call on the main AgentRuntime resource occurs between in-memory condition mutations and `Status().Update()`, the implementation MUST save and restore in-memory status to prevent silent data loss (per Constitution Principle I: Reconciler Status Integrity)
- **FR-009**: The `ControlPlaneMTLS` condition MUST transition from `PlainHTTP` to `mTLS` when SPIRE becomes available, without requiring controller restart
- **FR-010**: The `DataPlaneMTLS` condition MUST be mode-agnostic in naming (works for proxy-sidecar, envoy-sidecar, or any future authBridgeMode)

### Key Entities

- **ControlPlaneMTLS condition**: A Kubernetes status condition on AgentRuntime reporting the security posture of the controller-to-agent communication channel. Observed state, updated on each card fetch.
- **DataPlaneMTLS condition**: A Kubernetes status condition on AgentRuntime reporting the configured mTLS mode for agent-to-agent communication. Configured state, updated when the resolved configuration changes.
- **AgentRuntime status**: The existing `.status` field on the AgentRuntime custom resource, which already contains conditions for other operational states (e.g., Synced). The new conditions extend this existing set.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Platform engineers can determine the control-plane mTLS status of any agent by inspecting a single AgentRuntime field, without reading controller logs or tracing configuration
- **SC-002**: Platform engineers can determine the data-plane mTLS configuration of any agent by inspecting a single AgentRuntime field, without tracing through cluster defaults, namespace ConfigMaps, and CR overrides
- **SC-003**: When SPIRE availability changes, the control-plane condition reflects the new state within one reconciliation cycle
- **SC-004**: All plaintext fallback events are logged at warning level with sufficient context (agent name, namespace) for troubleshooting
- **SC-005**: Status conditions are queryable via standard tooling (e.g., filtering AgentRuntimes by condition status to find all agents not using mTLS)

## Assumptions

- The existing AgentRuntime status condition infrastructure (condition type, status, reason, message, lastTransitionTime) is sufficient and does not need extension
- Proxy-sidecar is the only authBridgeMode in scope for initial implementation, but condition names and semantics are designed to work for all modes
- The controller reconcile loop already triggers on agent card fetch events, so the `ControlPlaneMTLS` condition can be set within the existing reconcile path
- The resolved mTLS mode is already computed during sidecar injection (in the webhook injector), so `DataPlaneMTLS` can be derived from the same configuration chain
- SPIRE trust domain and verified fetch configuration are controller-level settings (flags), not per-agent settings
- The `AttestedAgentSpiffeID` field on AgentCard continues to store the SPIFFE ID from verified fetch; the `ControlPlaneMTLS` condition on AgentRuntime is a complementary signal, not a replacement
