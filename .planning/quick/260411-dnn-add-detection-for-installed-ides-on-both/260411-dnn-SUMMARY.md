---
phase: quick
plan: 260411-dnn
subsystem: detection
tags: [go, provider-detection, jetbrains, cross-platform]
requires:
  - phase: quick
    provides: existing provider detector registry and Result contracts
provides:
  - Cross-platform IDE install candidates for JetBrains IDE detectors on macOS and Linux
  - Language-aware JetBrains inference for phpstorm and goland keys
  - Regression tests for install path detection and inference gating by project signals
affects: [provider auto-detection, detect command output stability]
tech-stack:
  added: []
  patterns: [table-driven detector tests, deterministic candidate list evaluation]
key-files:
  created: [.planning/quick/260411-dnn-add-detection-for-installed-ides-on-both/260411-dnn-SUMMARY.md]
  modified: [internal/provider/detectors.go, internal/provider/detectors_test.go]
key-decisions:
  - "Use explicit per-provider candidate path lists to keep IDE detection deterministic and testable."
  - "Infer phpstorm/goland only when both JetBrains install evidence and matching language signal exist."
patterns-established:
  - "JetBrains language inference is additive fallback, not a replacement for direct app detection."
requirements-completed: [quick-cross-platform-ide-detection]
duration: 2min
completed: 2026-04-11
---

# Phase Quick Plan 260411-dnn: Add detection for installed IDEs on both Summary

**Cross-platform JetBrains IDE detection now supports macOS/Linux install conventions and infers PhpStorm/GoLand from project language signals when JetBrains is installed.**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-11T09:54:23Z
- **Completed:** 2026-04-11T09:56:28Z
- **Tasks:** 3/3
- **Files modified:** 2

## Accomplishments
- Added failing TDD regression coverage for Linux/macOS IDE install detection and JetBrains language inference behavior.
- Implemented deterministic cross-platform candidate path probing for JetBrains IDE providers.
- Added language-aware inference fallback so `composer.json` infers `phpstorm` and `go.mod` infers `goland` only when JetBrains install is detected.

## Task Commits

1. **Task 1: Add failing detector tests for Linux/macOS IDE installs and JetBrains language inference** - `67b4151` (test)
2. **Task 2: Implement cross-platform install probing and JetBrains language-aware IDE inference** - `316dda9` (feat)
3. **Task 3: Run focused provider + app detection regression suite** - no code changes (verification-only task)

## Files Created/Modified
- `internal/provider/detectors_test.go` - Adds deterministic regression tests for cross-platform IDE candidate matching and JetBrains language inference gating.
- `internal/provider/detectors.go` - Adds explicit IDE candidate lists, shared IDE detector helper, and language-aware inference fallback for `phpstorm`/`goland`.

## Decisions Made
- Used explicit, keyed candidate path lists (`ideInstallCandidatesByKey`) to support cross-platform detection while keeping output stable and test-injectable.
- Kept inference constrained to supported provider keys (`phpstorm`, `goland`) and explicit project signals to satisfy threat mitigation for false positives.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed global test fixture race in parallel subtests**
- **Found during:** Task 2
- **Issue:** New tests mutated global candidate-path fixtures while running in parallel, causing nondeterministic failures.
- **Fix:** Removed `t.Parallel()` from tests that override global candidate fixtures.
- **Files modified:** `internal/provider/detectors_test.go`
- **Verification:** `go test ./internal/provider -run "Test(IDE|JetBrains|AppDetector)"` passed consistently.
- **Committed in:** `316dda9`

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Required for deterministic tests and aligned with threat-model stability requirements.

## Issues Encountered
None.

## Threat Flags
None.

## Known Stubs
None.

## Self-Check: PASSED

- FOUND: `.planning/quick/260411-dnn-add-detection-for-installed-ides-on-both/260411-dnn-SUMMARY.md`
- FOUND: `67b4151`
- FOUND: `316dda9`
