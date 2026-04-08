# Project Research Summary

**Project:** genignore
**Domain:** Cross-platform Go CLI for deterministic `.gitignore` generation and safe file mutation
**Researched:** 2026-04-07
**Confidence:** HIGH

## Executive Summary

This project is a developer CLI in a mature category (template-based `.gitignore` generation), but the research points to a clear differentiation strategy: combine trusted template generation with deterministic, explainable auto-detection and safe managed-block updates. Experts build this type of tool with a thin command layer, pure domain logic for selection/planning, and isolated infrastructure adapters for filesystem + API interactions.

The recommended approach is opinionated: use Go 1.26.1 with Cobra + narrowly-scoped Viper, stdlib-first filesystem and HTTP integrations, and a plan-then-apply mutation flow that guarantees idempotent behavior. Ship v1 with table-stake generation/discovery, `detect` vs `add` semantics, deterministic ordering, and machine-readable JSON output; defer plugin systems, offline caching, and monorepo recursion until core reliability is proven.

The main risks are destructive file rewrites, non-atomic writes, API/provider drift, and noisy false-positive detection. Mitigation is straightforward and should be front-loaded: managed-marker ownership boundaries, atomic replace writes, runtime list reconciliation with the remote API, evidence-first detector outputs, and fixture-driven tests that avoid live network dependencies.

## Key Findings

### Recommended Stack

Research strongly supports a conservative, low-dependency Go stack that maximizes determinism and testability.

**Core technologies:**
- **Go 1.26.1**: runtime/toolchain — modern stable release with strong stdlib coverage for scanning, HTTP, JSON, and testing.
- **`spf13/cobra` v1.10.2**: CLI framework — de-facto standard for multi-command CLIs, help, and shell completions.
- **`spf13/viper` v1.21.0**: config binding — use only as a flag/env boundary adapter (avoid global config state).
- **`charmbracelet/lipgloss` v2.0.2**: terminal styling — meets human-output needs without full TUI complexity.

Critical version constraints: Go 1.26.1 compatibility with Cobra/Viper/staticcheck is validated; keep toolchain pins for reproducible CI.

### Expected Features

The feature research separates launch-critical requirements from differentiators and dangerous scope creep.

**Must have (table stakes):**
- Generate from single/multiple template keys in one run.
- Template discovery (`list`/`search`) using the same canonical provider namespace.
- Safe managed-block updates that preserve user-owned content.
- Deterministic output ordering and clear invalid-key handling.
- Cross-platform behavior and operational modes (`--dry-run`, `--verbose`, `--json`).

**Should have (competitive):**
- Automatic stack/environment detection with include/exclude controls.
- Explainable detection metadata (`why`, `source`, `path`) in verbose/JSON modes.
- Explicit `detect` (reset semantics) vs `add` (append semantics).
- API drift visibility between local assumptions and remote provider list.

**Defer (v2+):**
- Plugin ecosystem for detectors/providers.
- Monorepo recursive auto-management.
- Offline cache/sync subsystem and multi-source template merge engine.

### Architecture Approach

Architecture should follow clear boundaries: **CLI transport → app use-cases → pure domain logic → infrastructure adapters → output presenters**. This keeps Cobra-specific code thin and makes core behavior testable without filesystem/network side effects.

**Major components:**
1. **CLI adapters (`detect`, `add`)** — parse/validate flags and forward typed requests.
2. **Use-cases + planner** — orchestrate detection/selection/template fetch/file plan, supporting dry-run naturally.
3. **Domain modules** — deterministic provider selection, evidence model, and managed-block parse/render rules.
4. **Infrastructure adapters** — `.gitignore` repository with atomic writes, Toptal API client, environment probes.
5. **Presenters** — strict JSON contract plus concise human output with stdout/stderr discipline.

### Critical Pitfalls

1. **Destructive rewrites outside markers** — only mutate `BEGIN/END genignore` region; prove pre/post bytes are unchanged with fixtures.
2. **Non-atomic writes corrupting files** — stage+fsync+rename in same directory; never truncate-before-write.
3. **Provider/API drift and bad request construction** — reconcile against `/api/list`, normalize/sort keys, hard-fail on API errors, contract-test symbol keys (`c++`, `jetbrains+iml`).
4. **Over-aggressive detection false positives** — prioritize project evidence over global signals and expose evidence for every selected provider.
5. **Unscriptable output contracts** — JSON-only stdout in `--json`, stable schema, consistent non-zero exit codes for failure classes.

## Implications for Roadmap

Based on cross-file dependencies, use a 5-phase roadmap that front-loads safety and determinism before detection sophistication.

### Phase 1: Safe Mutation Foundation
**Rationale:** Data-loss risk is the highest-severity failure; safe writes and marker correctness must exist before feature expansion.
**Delivers:** Managed-block parser/renderer, plan model (create/insert/replace/noop), atomic file writer, invariance tests.
**Addresses:** Safe file update behavior, deterministic repeated runs.
**Avoids:** Destructive rewrites, non-atomic corruption.

### Phase 2: Template Integration and Deterministic Core
**Rationale:** Core product value is generation from authoritative templates with stable output.
**Delivers:** Toptal list/fetch client, key normalization/sorting, invalid-key warnings, list/search commands, template application.
**Uses:** Go stdlib HTTP/JSON + Cobra; typed errors and timeouts.
**Implements:** API adapter + domain selection boundaries.
**Avoids:** API drift/request bugs, ignore-semantics mishandling.

### Phase 3: Detection Engine + Explainability
**Rationale:** Differentiator should come after the safe deterministic core is proven.
**Delivers:** Detector registry (project/env/global), evidence-rich detection, `detect` semantics, include/exclude controls.
**Addresses:** Auto-detection and explainability differentiators.
**Avoids:** False positives and opaque provider selection.

### Phase 4: CLI UX and Automation Contract
**Rationale:** Once behavior stabilizes, lock interfaces consumed by humans and CI.
**Delivers:** `--dry-run`, `--verbose`, stable `--json` schemas, stdout/stderr separation, consistent exit code mapping.
**Addresses:** Automation readiness and trust.
**Avoids:** Unscriptable output and ambiguous failure handling.

### Phase 5: Reliability Hardening and Regression Safety
**Rationale:** Protect long-term maintainability before optional UX/features.
**Delivers:** `httptest` API contract suite, fixture/golden tests, testscript E2E scenarios, edge-case matrix (broken markers, permissions, timeouts).
**Addresses:** Release confidence for cross-platform CI.
**Avoids:** Live-network test flakiness and regressions in ordering/output shape.

### Phase Ordering Rationale

- Safety and idempotence dependencies force marker/atomic-write work first.
- API/template integration is prerequisite for both `add` and `detect` realism.
- Detection is strategically important but should not precede deterministic core correctness.
- Output contracts are stabilized after core behavior is fixed.
- Hardening last locks confidence while avoiding premature test brittleness.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2:** API contract edge-cases (encoding, platform newline behavior, failure taxonomy) merit focused validation.
- **Phase 3:** Detector signal weighting and false-positive control likely need targeted research and iteration.

Phases with standard patterns (can usually skip research-phase):
- **Phase 1:** Managed markers + atomic writes are established, well-documented patterns.
- **Phase 4:** CLI stdout/stderr and JSON contract conventions are standard.
- **Phase 5:** `httptest`/fixture/golden testing approaches are mature in Go CLI ecosystems.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Official docs/releases and strong ecosystem consensus; versions explicitly validated. |
| Features | HIGH | Anchored in official API docs and active competitor baselines (`gibo`, gitignore.io). |
| Architecture | HIGH | Aligns with established Go CLI layering and testability patterns. |
| Pitfalls | MEDIUM-HIGH | Most risks are well-supported, but some operational edge cases depend on implementation details and platform nuances. |

**Overall confidence:** HIGH

### Gaps to Address

- **Windows atomic-replace semantics:** confirm exact behavior and fallback strategy for cross-platform safety in implementation tests.
- **Detector calibration data:** no production telemetry yet; initial thresholds should be conservative and validated via fixture corpus.
- **Remote API resilience policy:** retries/timeouts/backoff limits require concrete SLO assumptions before finalization.
- **JSON schema versioning policy:** define how schema changes are introduced without breaking automation consumers.

## Sources

### Primary (HIGH confidence)
- Go stdlib docs (`io/fs`, `path/filepath#WalkDir`, `net/http`, `httptest`, `fstest`) — deterministic traversal, HTTP/testing foundations.
- Cobra and Viper official docs/pkg pages — command/flag architecture and config boundary patterns.
- gitignore.io/Toptal API docs — list/fetch endpoint behavior and template composition model.
- Git docs (`gitignore`, `git-check-ignore`) — ignore semantics and behavioral verification methods.

### Secondary (MEDIUM confidence)
- Active ecosystem repos (`gibo`, `go-internal/testscript`, `go-cmp`) — practical feature/testing baselines.
- `clig.dev` — CLI output and contract conventions.
- `github/gitignore` README — template taxonomy and curation norms.

### Tertiary (LOW confidence)
- Older/less-adopted tools with adjacent patterns (e.g., `gig`, newer multi-source wrappers) — useful signals but not primary design anchors.

---
*Research completed: 2026-04-07*
*Ready for roadmap: yes*
