# Review Guide: mTLS Status Conditions on AgentRuntime

**Generated**: 2026-06-02 | **Spec**: [spec.md](spec.md)

## Why This Change

The kagenti operator controller silently falls back to plaintext HTTP when SPIRE is unavailable for agent card fetches. There is no status signal on the AgentRuntime resource indicating whether mTLS or plaintext was used. Similarly, the resolved mTLS mode for agent-to-agent (data-plane) communication requires tracing through a three-layer configuration chain (cluster defaults, namespace ConfigMap, CR override) to determine the effective setting. Platform engineers operating agents at scale cannot audit the mTLS posture of their fleet without reading controller logs or manually resolving configuration.

## What Changes

Two new Kubernetes status conditions are added to AgentRuntime:

- **ControlPlaneMTLS**: reports whether the controller's most recent agent card fetch used mTLS (with the peer's SPIFFE ID) or fell back to plaintext HTTP. This is observed state, updated on each fetch.
- **DataPlaneMTLS**: reports the resolved mTLS mode for the agent's data-plane sidecar (strict, permissive, disabled, or no sidecar). This is configured state, derived from the existing config resolution chain.

Additionally, a warning-level structured log is emitted on each plaintext fallback, including the agent name and namespace. No breaking changes. No CRD schema changes (both conditions use the existing `[]metav1.Condition` field).

## How It Works

`ControlPlaneMTLS` is set inside the existing `fetchAndUpdateCard()` method in the AgentRuntime reconciler. The method already distinguishes between mTLS fetch (`AuthenticatedFetcher` with `FetchResult` containing the SPIFFE ID) and plaintext fallback (`AgentFetcher`). The condition is set immediately after the fetch result is known, before `persistCardFetchAnnotation()`. Since `persistCardFetchAnnotation()` does a metadata Patch that refreshes the local object (wiping in-memory status), the implementation saves and restores `rt.Status` across that call per Constitution Principle I.

`DataPlaneMTLS` is set in the main `Reconcile()` method after `ComputeConfigHash()` resolves the three-layer configuration. The resolved `AuthBridgeMode` and `MTLSMode` fields are read from the config result to determine the appropriate condition state.

Both conditions use the existing `setCondition()` helper that calls `meta.SetStatusCondition`, and are persisted by the single `Status().Update()` call at the end of the reconcile.

## When It Applies

**Applies when**:
- Agents are deployed with AgentRuntime CRs on clusters with or without SPIRE
- Platform engineers need to audit mTLS posture across their agent fleet
- Controller verified fetch is enabled (`--enable-verified-fetch` flag)
- Any authBridgeMode is configured (proxy-sidecar, envoy-sidecar, lite)

**Does not apply when**:
- Envoy-sidecar or lite modes (condition names are mode-agnostic but initial testing scope is proxy-sidecar only)
- Token exchange or OAuth2 flows (separate concern, outside mTLS transport scope)
- AgentCard conditions (existing Verified condition on AgentCard is unaffected)

## Key Decisions

1. **Two separate conditions instead of one composite**: Each condition answers a distinct question ("Is the controller talking securely?" vs "Are agents configured to talk securely?"). Allows independent alerting and clearer operational semantics. Alternative (single `TLSStatus` with compound reasons) was rejected because it conflates observed and configured state.

2. **Conditions on AgentRuntime, not AgentCard**: Keeps all mTLS status in one queryable resource. Alternative (control-plane status on AgentCard, where `AttestedAgentSpiffeID` already lives) was rejected because it splits mTLS visibility across two objects.

3. **Save/restore pattern for status integrity**: `persistCardFetchAnnotation()` does a metadata Patch that refreshes the local object. The implementation explicitly saves and restores `rt.Status` across this call to prevent silent condition loss. This follows the documented Constitution Principle I pattern that caught a production bug in this project.

4. **FetchSkipped as a condition state**: When the change-detection key matches and no fetch is performed, the condition is set to False/FetchSkipped rather than left unchanged. This ensures every reconcile produces an explicit condition state.

## Areas Needing Attention

- **Save/restore correctness**: The save/restore of `rt.Status` around `persistCardFetchAnnotation()` is the most critical implementation detail. The Patch call refreshes the entire object from the API server. If the save/restore is wrong, conditions will silently vanish (exactly the bug pattern from Constitution Principle I). The polish phase (T020) specifically validates this with an envtest read-back.

- **ConfigResult struct extension**: `DataPlaneMTLS` needs the resolved `AuthBridgeMode` and `MTLSMode` from `ComputeConfigHash()`. These fields are in the internal `resolvedConfig` but may not be exposed on the returned `ConfigResult`. Task T016 addresses this, but the reviewer should verify the struct extension doesn't break the config hash computation.

- **FetchSkipped vs last-known-good**: When a fetch is skipped (change key match), the condition shows False/FetchSkipped. This means a briefly healthy mTLS setup that stops fetching will show FetchSkipped rather than True/mTLS. This is intentional (the condition reflects the current reconcile's action, not cached state), but reviewers may question whether preserving the previous condition would be better.

## Open Questions

- Should the SPIFFE ID in the `ControlPlaneMTLS` condition message be the full SPIFFE URI (`spiffe://trust-domain/ns/default/sa/agent`) or just the workload identifier portion (`ns/default/sa/agent`)? Full URI is more explicit; short form is more readable.
- What happens to `ControlPlaneMTLS` when SPIRE is briefly unavailable during SPIRE agent pod restarts? The spec says to tolerate short-lived outages, but the implementation currently flips on each reconcile. A grace period could be added but adds complexity.

## Review Checklist

- [ ] Key decisions are justified
- [ ] Breaking changes are documented with migration guidance
- [ ] Scope matches the stated boundaries
- [ ] Success criteria are achievable
- [ ] No unstated assumptions
- [ ] Save/restore pattern correctly brackets the `persistCardFetchAnnotation()` Patch call
- [ ] Condition constants follow existing naming convention (`ConditionType*`)
- [ ] `ConfigResult` struct changes don't alter the config hash computation
- [ ] Warning log uses structured fields (agent name, namespace) not string formatting

---

<!-- Code phase sections are appended below this line by the phase-manager command -->
