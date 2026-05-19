# TODO

## Checklist

- [x] `15-02` Add compiled-binary end-to-end CLI coverage and reduce machine-dependent detection with shared detector inputs.
- [x] `15-03` Add fuzzing and benchmark coverage for managed-block rewriting and detector-performance-sensitive paths.
- [x] `15-04` Add release or install smoke tests and document deliberate toolchain and dependency refresh workflows.
- [ ] `16-01` Pin upstream template sourcing by commit SHA and tighten cache freshness, integrity, and conditional fetch behavior.
- [ ] `16-02` Add explain or doctor visibility, managed-block provenance, and a real diff-before-write flow.
- [ ] `16-03` Build a real-repository fixture corpus and add machine-readable output-stability contract tests.
- [ ] `16-04` Separate detection and provider resolution from file mutation for scripting and document the narrow deferred preset boundary.

## Execution Order

1. `15-02`
2. `15-03`
3. `15-04`
4. `16-01`
5. `16-02`
6. `16-03`
7. `16-04`

## 15-02

### Summary

Increase confidence in the real CLI binary while making detection steadier and cheaper.

### Why

- Catch packaging and subprocess regressions.
- Reduce host-dependent output variance.
- Avoid repeated detector rescans.

### Depends On

- `15-01` complete

### Requirements

- `R21`
- `R22`
- `R23`

### Key Files

- `internal/provider/detectors.go`
- `internal/provider/detectors_test.go`
- `internal/app/service.go`
- `internal/app/service_test.go`
- `README.md`
- `testdata/`

### Tasks

- Add compiled-binary end-to-end CLI coverage.
- Build or invoke the compiled `genignore` binary in tests.
- Run representative `detect`, `add`, `list`, or help scenarios against fixture repositories.
- Assert real stdout or stderr, exit status, and managed-file outcomes.
- Reduce machine-dependent detection behavior.
- Audit PATH, host OS, and local app install signals.
- Prefer repository-backed evidence where possible.
- Precompute shared detector input once per command so detectors stop rescanning the same top-level directories.

### Acceptance Criteria

- Binary-level tests execute the real CLI subprocess.
- Output and exit behavior are asserted for representative user flows.
- Equivalent repos produce more stable provider sets across host differences.
- Shared detector inputs reduce repeated rescans without changing traversal scope.

### Verification

- `go test ./internal/app`
- `go test ./internal/provider ./internal/app`

### Deliverable

- `.planning/phases/phase-15-post-v1-runtime-resilience-and-confidence-hardening/15-02-SUMMARY.md`

## 15-03

### Summary

Harden the managed-block rewrite path with adversarial coverage and performance baselines.

### Why

- Catch parser and rewrite edge cases earlier.
- Make rewrite performance tradeoffs measurable.

### Depends On

- `15-02` complete

### Requirements

- `R24`

### Key Files

- `internal/gitignore/manager.go`
- `internal/gitignore/manager_test.go`
- `internal/gitignore/fuzz_test.go`
- `internal/gitignore/benchmark_test.go`
- `README.md`
- `.planning/codebase/CONCERNS.md`

### Tasks

- Add Go fuzz targets for marker parsing and managed-block replacement.
- Cover malformed, truncated, duplicated, and adversarial `.gitignore` inputs.
- Distill any interesting fuzz findings into fixed regression tests.
- Add benchmarks for create, update, and no-op rerun paths.
- Document any benchmark-driven constraints or usage guidance.

### Acceptance Criteria

- Fuzz targets cover representative malformed inputs.
- Known interesting cases become stable regression tests.
- Representative rewrite paths have named benchmarks.
- Contributors can run the benchmarks intentionally.

### Verification

- `go test ./internal/gitignore`
- `go test -run '^$' -bench . ./internal/gitignore`

### Deliverable

- `.planning/phases/phase-15-post-v1-runtime-resilience-and-confidence-hardening/15-03-SUMMARY.md`

## 15-04

### Summary

Extend release confidence from packaging validation to installable artifact smoke tests, then document deliberate version-refresh workflow.

### Why

- Catch release-only install regressions earlier.
- Make maintenance updates explicit, reviewable, and repeatable.

### Depends On

- `15-03` complete

### Requirements

- `R25`
- `R26`

### Key Files

- `.github/workflows/ci.yml`
- `.github/workflows/release-please.yml`
- `.goreleaser.yaml`
- `README.md`
- `AGENTS.md`
- `.planning/codebase/CONCERNS.md`

### Tasks

- Add release or install smoke tests for packaged artifacts.
- Extend pre-publish validation so CI installs or unpacks packaged artifacts.
- Run a minimal real command from packaged output.
- Document the toolchain and dependency refresh workflow.
- Spell out how Go, GoReleaser, and key pinned dependencies are reviewed or refreshed.
- Keep version bump work narrow and explicitly verified.

### Acceptance Criteria

- At least one packaged install path is exercised automatically.
- The smoke test runs a real command from packaged output.
- Contributor guidance names the expected refresh and verification steps.
- Version changes are framed as isolated maintenance work rather than incidental drift.

### Verification

- `rg -n 'goreleaser|smoke|install|artifact' .github/workflows/ci.yml .github/workflows/release-please.yml .goreleaser.yaml README.md`
- `rg -n 'refresh|upgrade|toolchain|dependency|GoReleaser|version' README.md AGENTS.md .planning/codebase/CONCERNS.md`

### Deliverable

- `.planning/phases/phase-15-post-v1-runtime-resilience-and-confidence-hardening/15-04-SUMMARY.md`

## 16-01

### Summary

Make remote template sourcing reproducible and cache behavior more trustworthy.

### Why

- Let users and releases regenerate against a known upstream revision.
- Make cache freshness and degraded-mode rules explicit.

### Depends On

- `15-01` complete

### Requirements

- `R27`
- `R31`
- `R33`

### Key Files

- `internal/api/client.go`
- `internal/api/client_test.go`
- `internal/app/catalog.go`
- `internal/app/catalog_test.go`
- `internal/provider/supported.go`
- `README.md`
- `AGENTS.md`

### Tasks

- Pin upstream catalog and template sourcing by commit SHA.
- Keep the embedded-exception boundary unchanged.
- Validate cache integrity and freshness explicitly.
- Add conditional HTTP fetch reuse via stored `ETag` metadata.
- Reuse cached content on `304 Not Modified`.

### Acceptance Criteria

- Pinning is explicit and reviewable.
- Equivalent inputs against the same pinned revision remain byte-stable.
- Corrupt or stale cache states fail clearly or refresh predictably.
- Conditional requests are covered without live network calls.

### Verification

- `go test ./internal/api ./internal/app ./internal/provider`
- `go test ./internal/api ./internal/app`

### Deliverable

- `.planning/phases/phase-16-reproducibility-observability-and-scriptable-safety-hardening/16-01-SUMMARY.md`

## 16-02

### Summary

Improve user visibility and pre-write confidence around managed `.gitignore` changes.

### Why

- Make runtime decisions inspectable.
- Show the exact diff before writes.
- Explain generated content through deterministic provenance.

### Depends On

- `15-01` complete
- `15-02` complete

### Requirements

- `R28`
- `R29`
- `R30`

### Key Files

- `internal/app/cli.go`
- `internal/app/cli_test.go`
- `internal/app/service.go`
- `internal/app/service_test.go`
- `internal/gitignore/manager.go`
- `internal/gitignore/manager_test.go`
- `README.md`

### Tasks

- Add an `explain` or `doctor` style read-only diagnostics surface.
- Surface detector evidence, provider resolution, cache state, and degraded-runtime decisions.
- Add deterministic provenance metadata inside the managed block.
- Add a real diff-before-write command or flag.
- Keep no-op, create, and update reporting aligned with the actual write path.

### Acceptance Criteria

- Diagnostics do not require internal logging knowledge.
- Output distinguishes repository evidence from host-only heuristics where applicable.
- Provenance lines stay byte-stable for equivalent inputs.
- Diff preview matches the eventual write result exactly.

### Verification

- `go test ./internal/app ./internal/provider`
- `go test ./internal/gitignore ./internal/app`
- `go test ./internal/app ./internal/gitignore ./internal/provider`

### Deliverable

- `.planning/phases/phase-16-reproducibility-observability-and-scriptable-safety-hardening/16-02-SUMMARY.md`

## 16-03

### Summary

Raise confidence with realistic fixtures and explicit stability contracts.

### Why

- Reduce the chance that detector or managed-output regressions slip through because tests only cover toy repo shapes.

### Depends On

- `15-02` complete
- `15-03` complete
- `16-01` complete
- `16-02` complete

### Requirements

- `R32`
- `R34`

### Key Files

- `internal/app/cli_test.go`
- `internal/app/service_test.go`
- `internal/provider/detectors_test.go`
- `internal/gitignore/manager_test.go`
- `testdata/repos/`
- `README.md`
- `AGENTS.md`

### Tasks

- Build a curated fixture corpus from real repositories.
- Keep fixtures reduced, reviewable, secret-free, and CI-sized.
- Use fixtures to exercise representative detector and provider-selection scenarios.
- Add machine-readable output-stability contract fixtures or snapshots.
- Make intentional contract updates explicit and easy to review.

### Acceptance Criteria

- Fixture provenance and normalization rules are documented.
- Real repository shapes are covered beyond synthetic tests.
- Contract files are easy to diff.
- Ordering, formatting, or provenance churn is caught automatically.

### Verification

- `go test ./internal/provider ./internal/app`
- `go test ./internal/gitignore ./internal/app`
- `go test ./internal/provider ./internal/gitignore ./internal/app`

### Deliverable

- `.planning/phases/phase-16-reproducibility-observability-and-scriptable-safety-hardening/16-03-SUMMARY.md`

## 16-04

### Summary

Make `genignore` easier to automate without widening the product into per-project configuration.

### Why

- Expose a supported read-only scripting surface.
- Keep future preset ideas explicitly out of current scope.

### Depends On

- `15-02` complete
- `16-02` complete

### Requirements

- `R35`
- `R36`

### Key Files

- `internal/app/cli.go`
- `internal/app/cli_test.go`
- `internal/app/service.go`
- `internal/app/service_test.go`
- `README.md`
- `.planning/PROJECT.md`
- `AGENTS.md`

### Tasks

- Add a supported read-only command or flag for detection or resolved-provider output.
- Reuse the same ordering, validation, and normalization pipeline as mutating flows.
- Keep scripting output machine-readable and deterministic.
- Document that the scripting and diagnostics surface is not per-project configuration.
- Keep any preset concept explicitly deferred.

### Acceptance Criteria

- Detection and provider resolution can be invoked without file mutation.
- Read-only and mutating paths stay aligned on provider resolution.
- Docs do not imply per-project preset support exists now.
- No new command or file contract becomes a de facto preset system.

### Verification

- `go test ./internal/app`

### Deliverable

- `.planning/phases/phase-16-reproducibility-observability-and-scriptable-safety-hardening/16-04-SUMMARY.md`
