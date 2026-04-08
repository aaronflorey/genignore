# Phase 5: Reliability & Regression Safety - Context

**Gathered:** 2026-04-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver deterministic, offline-safe regression coverage for shipped v1 behavior across API fixtures, detection and selection semantics, `.gitignore` mutation safety, dry-run behavior, and output contracts, fixing any real product bugs that the new regression tests expose.

</domain>

<decisions>
## Implementation Decisions

### Regression Surface
- Focus phase 5 on shipped user-facing behaviors: file migration, selection semantics, warnings, JSON/output contracts, and failure handling.
- Add regression coverage for real issues surfaced in review during this phase and fix behavior where the tests expose actual bugs.
- Prefer targeted regression tests around command and service behavior, using lower-level tests only where they materially close a coverage gap.
- Treat ordering, warning order, JSON field presence, and no-op reruns as contract-level deterministic behavior.

### Fixture Strategy
- Keep API-dependent tests fully offline with checked-in fixtures and `httptest` servers.
- Make fixtures realistic enough to catch remote contract drift, especially provider-list response shape changes.
- Use temp directories for filesystem scenarios rather than a separate external integration harness by default.
- Extend the current Go test suite first; only introduce a heavier harness if a concrete remaining gap forces it.

### Detection And Selection Regressions
- Add explicit regression coverage for false positives and structured detector errors, including the current Vue-on-Vite false positive risk.
- Lock `detect` reset semantics, `add` append-only behavior, `detected + include - exclude`, and deterministic ordering.
- Assert unsupported-key and remote-provider drift warnings explicitly and deterministically.
- Keep regression coverage for sorted `detectionResults` and stable evidence visibility across default, verbose, and JSON output.

### File Safety And No-Op Behavior
- Cover missing file, existing file without markers, existing file with markers, malformed markers, and dry-run on existing files.
- Add regression coverage proving equivalent reruns avoid unnecessary rewrites while keeping results stable.
- Verify provider-list and template API failures do not write `.gitignore`.
- Keep tests proving all user-owned content outside the managed block, including leading blank lines, remains byte-exact.

### the agent's Discretion
The agent has discretion over test organization, fixture placement, and whether a small shared test helper is warranted, as long as the resulting regression suite stays offline-safe, deterministic, and aligned with shipped behavior rather than speculative future features.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/service_test.go`, `internal/app/cli_test.go`, `internal/app/json_test.go`, `internal/gitignore/manager_test.go`, `internal/provider/detectors_test.go`, and `internal/api/client_test.go` already provide the main regression surface.
- `internal/api/testdata/` already contains checked-in API fixtures and `internal/api/client_test.go` already uses `httptest`.
- The CLI, service, detector, and manager packages are already separated cleanly enough to add focused regression tests without a new architecture.

### Established Patterns
- Use temp directories for filesystem behavior.
- Use typed structs and deterministic ordering assertions rather than loose string matching where possible.
- Keep tests offline and package-local.
- Verify side effects explicitly, especially around `.gitignore` writes and file preservation.

### Integration Points
- Expand `internal/api/client_test.go` fixtures and assertions to match the real provider-list contract.
- Add detector-regression cases in `internal/provider/detectors_test.go`.
- Strengthen no-op and safety behavior in `internal/gitignore/manager.go` and `internal/gitignore/manager_test.go` if regression tests expose churn or boundary bugs.
- Extend `internal/app/service_test.go`, `internal/app/cli_test.go`, and `internal/app/json_test.go` to lock end-to-end selection and output contracts.

</code_context>

<specifics>
## Specific Ideas

- Use this phase to close the gap between currently passing tests and the real v1 product contract, especially around realistic API fixtures and no-op file churn.
- Keep any behavior fixes tightly coupled to regression tests so the final milestone ends with a trustworthy suite.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope.

</deferred>
