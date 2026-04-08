# Roadmap: genignore

## Overview

Deliver a safe, deterministic CLI in dependency order: first guarantee `.gitignore` mutation safety, then add template/API-driven provider management, then ship auto-detection controls, then lock automation-facing output contracts, and finally harden reliability with fixture-driven regression coverage.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Managed Block Safety** - Safe, idempotent `.gitignore` ownership inside markers only.
- [x] **Phase 2: Provider Selection & API Generation** - Deterministic provider selection, discovery, and template generation.
- [x] **Phase 3: Auto-Detection Workflow** - Signal-based detection with include/exclude controls and evidence.
- [x] **Phase 4: Output & Automation Contracts** - Stable human output, JSON contract, and dry-run behavior.
- [x] **Phase 5: Reliability & Regression Safety** - Deterministic fixture coverage for user-facing behavior and failures. (completed 2026-04-08)

## Phase Details

### Phase 1: Managed Block Safety
**Goal**: Users can regenerate `.gitignore` repeatedly without losing any content outside managed markers.
**Depends on**: Nothing (first phase)
**Requirements**: GIT-01, GIT-02, GIT-03, GIT-04, GIT-05
**Success Criteria** (what must be TRUE):
  1. User can run the CLI in a repo with no `.gitignore` and get a new file containing a managed block.
  2. User can run the CLI in a repo with an existing `.gitignore` and all non-managed content remains unchanged.
  3. User can run the CLI when markers already exist and only content between markers is replaced.
  4. User sees equivalent repeated runs produce no non-deterministic file churn.
**Plans**:
  - Plan 1: Deterministic Managed Block Safety (`.planning/phases/1-managed-block-safety/2-PLAN.md`)

### Phase 2: Provider Selection & API Generation
**Goal**: Users can choose valid providers, discover available keys, and generate templates deterministically from the Toptal API.
**Depends on**: Phase 1
**Requirements**: CMD-05, CMD-06, DET-03, DET-04, API-01, API-02, API-03
**Success Criteria** (what must be TRUE):
  1. User can run `genignore add <keys...>` and only missing valid providers are appended to the managed set.
  2. User can list/search supported provider keys before generating.
  3. User sees unsupported keys reported as warnings while valid keys still proceed.
  4. User gets alphabetically ordered providers in file metadata, API requests, and command outputs.
  5. User gets generated template content from Toptal API, with hard failure on API errors and warning when remote list drifts from hardcoded support.
**Plans**: 2 plans

Plans:
- [x] 02-01-PLAN.md — Harden deterministic add/detect provider + API behavior with regression coverage
- [x] 02-02-PLAN.md — Add deterministic provider discovery commands (`list`/`search`)

### Phase 3: Auto-Detection Workflow
**Goal**: Users can auto-detect providers from local context and deliberately adjust final selection before generation.
**Depends on**: Phase 2
**Requirements**: CMD-01, CMD-02, CMD-03, CMD-04, DET-01, DET-02
**Success Criteria** (what must be TRUE):
  1. User can run `genignore detect` and receive providers inferred from project files/folders, OS, and installed software signals.
  2. User can run `detect --include` and `detect --exclude` to force-add or remove providers from the detected set before generation.
  3. User sees a hard error when `detect` resolves to an empty final provider set.
  4. User can inspect detection evidence (`reason`, `source`, `path` when available) in verbose or JSON output.
**Plans**: 2 plans

Plans:
- [x] 03-01-PLAN.md — Harden deterministic detect selection, detector evidence, and empty-result behavior
- [x] 03-02-PLAN.md — Expose detection evidence cleanly in verbose and JSON CLI output

### Phase 4: Output & Automation Contracts
**Goal**: Users can trust both human-readable and machine-readable outputs for local use and CI automation.
**Depends on**: Phase 3
**Requirements**: OUT-01, OUT-02, OUT-03
**Success Criteria** (what must be TRUE):
  1. User gets concise human-readable output summarizing selection changes, warnings, and file action.
  2. User can run `--json` and receive a structured execution payload including command context, selections, warnings, detection reasons, and file action.
  3. User can run `--dry-run` and preview intended changes without modifying `.gitignore`.
**Plans**: 2 plans

Plans:
- [x] 04-01-PLAN.md — Lock detect/add JSON contract and dry-run behavior for automation
- [x] 04-02-PLAN.md — Refine concise CLI summaries and stable list/search output contracts

### Phase 5: Reliability & Regression Safety
**Goal**: Maintainers can validate all shipped user behaviors with deterministic, offline-safe regression coverage.
**Depends on**: Phase 4
**Requirements**: TST-01, TST-02, TST-03, TST-04
**Success Criteria** (what must be TRUE):
  1. Contributor can run fixture-based tests without live API dependency and get deterministic pass/fail outcomes.
  2. Test suite verifies `.gitignore` migration behaviors for missing files, existing files without markers, and existing files with markers.
  3. Test suite verifies selection semantics (`detect` reset, `add` append-only), ordering stability, and invalid-key warning behavior.
  4. Test suite verifies failure/output contracts including API failure handling, dry-run non-write behavior, and JSON shape stability.
**Plans**: 2 plans

Plans:
- [x] 05-01-PLAN.md — Lock offline API fixtures and `.gitignore` file-safety regressions
- [x] 05-02-PLAN.md — Harden detector, selection, warning, and output regressions end to end

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Managed Block Safety | 1/1 | Completed | 2026-04-07 |
| 2. Provider Selection & API Generation | 2/2 | Completed | 2026-04-07 |
| 3. Auto-Detection Workflow | 2/2 | Completed | 2026-04-08 |
| 4. Output & Automation Contracts | 2/2 | Completed | 2026-04-08 |
| 5. Reliability & Regression Safety | 2/2 | Complete    | 2026-04-08 |
