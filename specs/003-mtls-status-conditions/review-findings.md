# Deep Review Findings

**Date:** 2026-06-02
**Branch:** 003-mtls-status-conditions
**Rounds:** 1
**Gate Outcome:** PASS
**Invocation:** manual

## Summary

| Severity | Found | Fixed | Remaining |
|----------|-------|-------|-----------|
| Critical | 1 | 1 | 0 |
| Important | 5 | 5 | 0 |
| Minor | 5 | - | 5 |
| **Total** | **11** | **6** | **5** |

**Agents completed:** 5/5 (+ 1 external tool)
**Agents failed:** none

## Findings

### FINDING-1
- **Severity:** Critical
- **Confidence:** 95
- **File:** agentruntime_controller.go:755-756
- **Category:** correctness
- **Source:** correctness-agent (also reported by: architecture-agent, coderabbit, production-agent)
- **Round found:** 1
- **Resolution:** fixed (round 1)

**What is wrong:**
ControlPlaneMTLS condition was unconditionally set to `ConditionFalse` with reason "FetchSkipped" when the card fetch was skipped due to an unchanged change key. This overwrote a previous `True/mTLS` condition on every subsequent reconcile after a successful mTLS fetch.

**Why this matters:**
Violates FR-001 and FR-009. The condition would flip from True to False on every reconcile after the first mTLS success, making it useless for auditing mTLS posture. Monitoring systems would see constant flapping.

**How it was resolved:**
Removed the unconditional `setCondition` call from the FetchSkipped path. The existing ControlPlaneMTLS condition from the previous reconcile is preserved in `rt.Status.Conditions` since the status is not reset between reconciles.

**External tool analysis (CodeRabbit):**
> The current fetch-skipped path unconditionally calls r.setCondition(rt, ConditionTypeControlPlaneMTLS, metav1.ConditionFalse, "FetchSkipped", ...) which overwrites a previously true mTLS status; change this to preserve the existing ConditionTypeControlPlaneMTLS when fetch is skipped.

### FINDING-2
- **Severity:** Important
- **Confidence:** 90
- **File:** agentruntime_config.go:181-182
- **Category:** correctness
- **Source:** correctness-agent
- **Round found:** 1
- **Resolution:** fixed (round 1)

**What is wrong:**
`resolveConfig` set `AuthBridgeMode` and `MTLSMode` from only the CR spec, without falling back to the namespace-level `authbridge-runtime-config` ConfigMap. The webhook injector correctly implements a fallback chain (CR override > namespace YAML > empty default), but the controller's resolution did not match.

**Why this matters:**
Violates FR-005. When an operator sets `mtlsMode: strict` in the namespace ConfigMap but leaves the CR field empty, the DataPlaneMTLS condition would incorrectly report "Disabled".

**How it was resolved:**
Added `extractModeFromYAML` and `extractMTLSModeFromYAML` helper functions in `agentruntime_config.go` (mirroring `injector.ExtractMode`/`ExtractMTLSMode` to avoid circular imports). Added fallback logic: if the CR spec field is empty but the namespace ConfigMap has a value, use the ConfigMap value.

### FINDING-3
- **Severity:** Important
- **Confidence:** 95
- **File:** agentruntime_controller_test.go:1161-1196
- **Category:** test-quality
- **Source:** test-agent (also reported by: architecture-agent)
- **Round found:** 1
- **Resolution:** fixed (round 1)

**What is wrong:**
Test name said "should set False/Disabled when AuthenticatedFetcher is nil" but assertion checked for reason `PlainHTTP`. The actual `Disabled` path (when `EnableCardDiscovery == false`) had no test coverage.

**Why this matters:**
Spec acceptance scenario 3 ("verified fetch is not enabled -> Disabled") was untested. If the implementation's `Disabled` branch were deleted, no test would fail.

**How it was resolved:**
Renamed the existing test to "should set False/PlainHTTP when AuthenticatedFetcher is nil". Added a new test "should set False/Disabled when card discovery is disabled" that sets `EnableCardDiscovery: false` and verifies `cond.Reason == "Disabled"`.

### FINDING-4
- **Severity:** Important
- **Confidence:** 90
- **File:** agentruntime_controller_test.go:1380-1431
- **Category:** test-quality
- **Source:** test-agent
- **Round found:** 1
- **Resolution:** fixed (round 1)

**What is wrong:**
Warning log test used `funcr.Options{Verbosity: 1}` which captures all log levels including V(1). A regression changing the log from V(0) to V(2) would still pass.

**Why this matters:**
FR-007 specifically requires "warning-level log" (V(0) in logr). If verbosity is raised, operators would not see the fallback warning in production logs.

**How it was resolved:**
Changed to `funcr.Options{Verbosity: 0}` which only captures V(0) messages, ensuring the test would fail if the log level changed.

### FINDING-5
- **Severity:** Important
- **Confidence:** 85
- **File:** agentruntime_controller_test.go:1433-1535
- **Category:** test-quality
- **Source:** test-agent
- **Round found:** 1
- **Resolution:** fixed (round 1)

**What is wrong:**
Transition test had dead code (unused `failFetcher` variable and comments explaining the pivot from SPIRE failure to port toggle). Test name was misleading.

**Why this matters:**
Dead comments reduce confidence in test design. The unused variable clutters the code.

**How it was resolved:**
Removed dead code and the unused variable. Renamed test to "should transition from PlainHTTP to mTLS when TLS port becomes available" to accurately describe what is tested.

### FINDING-6
- **Severity:** Important
- **Confidence:** 90
- **File:** agentruntime_controller_test.go (multiple)
- **Category:** test-quality
- **Source:** test-agent
- **Round found:** 1
- **Resolution:** fixed (round 1)

**What is wrong:**
PlainHTTP, Disabled, and FetchSkipped tests omitted message assertions. FR-008 requires conditions to follow Kubernetes convention including message content.

**Why this matters:**
A bug setting reason=PlainHTTP with the wrong message would pass undetected.

**How it was resolved:**
Added `Expect(cond.Message).To(ContainSubstring(...))` assertions to the PlainHTTP and Disabled tests. Updated the FetchSkipped test to verify condition preservation semantics.

### FINDING-7 (Minor, remaining)
- **Severity:** Minor
- **Confidence:** 80
- **File:** agentruntime_controller.go:712-727
- **Category:** architecture
- **Source:** architecture-agent
- **Resolution:** remaining

**What is wrong:**
Unrecognized `MTLSMode` values (e.g., typos like "strickt") silently result in `Disabled` with no warning.

**Why this matters:**
A user typo silently downgrades security posture.

### FINDING-8 (Minor, remaining)
- **Severity:** Minor
- **Confidence:** 75
- **File:** agentruntime_controller.go:741-742
- **Category:** architecture
- **Source:** architecture-agent
- **Resolution:** remaining

**What is wrong:**
Message says "status unknown" but condition status is `False`, which asserts "mTLS is not active".

### FINDING-9 (Minor, remaining)
- **Severity:** Minor
- **Confidence:** 70
- **File:** agentruntime_controller.go:776-777
- **Category:** security
- **Source:** security-agent
- **Resolution:** remaining

**What is wrong:**
SPIFFE ID in condition message provides a second exposure path. Theoretical information leak concern.

### FINDING-10 (Minor, remaining)
- **Severity:** Minor
- **Confidence:** 80
- **File:** agentruntime_controller.go:760-772
- **Category:** correctness
- **Source:** correctness-agent
- **Resolution:** remaining

**What is wrong:**
ControlPlaneMTLS condition not set on error paths (service not found, fetch failed). Stale True condition could persist during failures.

### FINDING-11 (Minor, remaining)
- **Severity:** Minor
- **Confidence:** 80
- **File:** agentruntime_controller_test.go
- **Category:** test-quality
- **Source:** test-agent
- **Resolution:** remaining

**What is wrong:**
No negative test verifying warning log is NOT emitted when mTLS succeeds.

## Test Suite Results

| Round | Test Command | Exit Code | Failures | Status |
|-------|-------------|-----------|----------|--------|
| 1     | go test ./... | 0 | 0 | passed |

Test suite passed in all fix rounds. E2e tests excluded (require Docker, pre-existing failure).
