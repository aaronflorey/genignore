# genignore

## What This Is

`genignore` is a Go CLI that detects relevant `.gitignore` templates for the current project and environment, fetches merged rules from the Toptal gitignore API, and manages only a generated block inside `.gitignore`. It is built for developers who want safe, repeatable ignore generation without losing manual rules outside managed markers.

## Core Value

Generate and maintain a deterministic, safe managed `.gitignore` block that users can run repeatedly without losing their own file content.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] `detect` command resets managed providers from `detected + include - exclude` and errors on empty result
- [ ] `add` command appends only missing providers and rebuilds managed block from resulting set
- [ ] Managed block updates are idempotent and preserve all user content outside markers
- [ ] Provider detection combines project, environment, and installed software signals with rich evidence metadata
- [ ] Provider keys are sorted alphabetically before storage, API fetch, and outputs
- [ ] API integration fetches provider list and template content; API failures hard-fail execution
- [ ] Default output is concise human-readable; `--json` exposes structured execution metadata
- [ ] `--dry-run` shows intended changes without writing `.gitignore`
- [ ] Fixture-based tests cover file migrations, selection behavior, ordering, warnings, JSON shape, and API failures

### Out of Scope

- Monorepo-aware detection/management in v1 — keep implementation scoped to current working directory
- Plugin system for external detectors/providers — defer until core detection architecture is validated
- Config file support — avoid adding persistent configuration in v1
- Offline mode or local template fallback — API unavailability should fail clearly
- Persistent cache beyond process lifetime — unnecessary for v1 behavior

## Context

The project is a greenfield CLI with a locked v1 product definition in `PRD.md`. The implementation stack is Go with Cobra/Viper for command handling and Charmbracelet libraries for terminal UX. Provider support is hardcoded in v1 using a known provider list sourced from Toptal API metadata and `github/gitignore` references. Detection intentionally mixes local project signals (files/folders), runtime environment signals (OS), and global software signals (installed tools/IDEs).

## Constraints

- **Tech stack**: Go + Cobra/Viper + Charmbracelet output — align with requested v1 tooling
- **API dependency**: Toptal gitignore API is required at runtime — no local fallback in v1
- **Safety**: Only content between managed markers is CLI-owned — user lines outside markers must remain untouched
- **Determinism**: Provider ordering must be alphabetical — avoid output/file churn across runs
- **Scope**: Current directory only — no monorepo traversal or plugin loading in v1

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Use `detect` for full reset and `add` for append-only behavior | Separates automatic project inference from explicit manual extension | — Pending |
| Hardcode supported providers in binary while checking remote list at runtime | Keeps v1 predictable while still surfacing provider drift warnings | — Pending |
| Use managed markers in `.gitignore` and overwrite only managed region | Guarantees idempotent regeneration and preserves user-owned content | — Pending |
| Keep API failures as hard failures | Prevents silently stale or partial ignore content | — Pending |
| Separate `Provider` detection from `Manager` file/API orchestration | Keeps detection extensible and side-effectful logic centralized | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-07 after initialization*
