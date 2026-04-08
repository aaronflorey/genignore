# Phase 2: Provider Selection & API Generation - Research

**Researched:** 2026-04-07  
**Domain:** Deterministic provider selection + Toptal API integration + provider discovery commands  
**Confidence:** HIGH

## User Constraints

No `*-CONTEXT.md` exists for this phase, so there are no extra locked UI/UX decisions beyond roadmap + requirements.

### Locked Decisions
- None from discuss-phase context (file absent).

### the agent's Discretion
- Command shape for provider discovery (`list`, `search`, or both) as long as `CMD-06` is fully satisfied.
- Exact placement of provider-discovery logic (`service` method vs dedicated catalog use-case).

### Deferred / Out of Scope
- Monorepo traversal, plugin loading, offline cache/fallback, project config file (v1 out-of-scope constraints).

## Requirement-to-Code Status (Current Snapshot)

| Requirement | Status in current code | Gap to close in planning |
|-------------|------------------------|---------------------------|
| CMD-05 (`add` append-only missing valid providers) | Implemented in `internal/app/service.go::Add` + tested (`TestAddAppendsOnlyMissingProviders`) | Strengthen coverage for warning + ordering edge cases across CLI behavior |
| CMD-06 (discover keys via list/search command(s)) | **Missing command surface** in `internal/app/cli.go` | Add explicit `list`/`search` command path and tests |
| DET-03 (warn on unsupported while continuing valid keys) | Implemented via `sanitizeKeys` warnings in `Detect`/`Add` | Ensure deterministic warning order and command-level output checks |
| DET-04 (alphabetical determinism in metadata/API/output) | Implemented for final provider set (`mapKeysSorted`) and add-result sorting | Ensure provider discovery output and API list usage are sorted deterministically |
| API-01 (template generation from final set) | Implemented (`FetchTemplate` in `Detect`/`Add`) | Preserve and verify request uses sorted final set |
| API-02 (API failures hard-fail) | Implemented (errors returned directly from API calls) | Add coverage for list/search command failure semantics |
| API-03 (warn when hardcoded provider absent remotely) | Implemented (`remoteDiffWarnings`) | Verify this warning appears in all relevant command paths |

## Key Technical Findings

1. **Core phase behavior already exists for `detect`/`add`**, including warnings and remote drift checks.
2. **Phase-completion blocker is command discoverability (`CMD-06`)** — no list/search command currently registered on root Cobra command.
3. **Determinism is strong but must be applied to discovery output too** (not just `FinalProviders`).
4. **API client contract is straightforward and testable with `httptest` fixtures**, already used in `internal/api/client_test.go`.

## Recommended Implementation Strategy

### 1) Add provider discovery command(s)
- Preferred: implement both `list` (all supported keys) and `search <term>` (filtered subset).
- Use one shared provider-catalog query path so command behavior and tests stay consistent.
- Enforce sorted output before rendering to human/JSON modes.

### 2) Keep provider-set determinism single-sourced
- Continue using sorted slices for all externally visible provider lists.
- Verify API template requests receive sorted provider keys.
- Ensure warnings are stable/sorted when multiple keys are invalid.

### 3) Expand tests where phase requirements can regress
- Add unit tests for discovery commands and search filtering.
- Add tests covering deterministic warning ordering and API-hard-failure in discovery path.
- Keep API tests offline using `httptest` and fixture files.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Discovery command returns unsorted or inconsistent keys | Violates DET-04 and causes noisy automation output | Sort at source of discovery response and assert sorted order in tests |
| Search/list introduces a second provider source that drifts from `SupportedKeys` | Users see keys that cannot be used by add/detect | Reuse `provider.SupportedKeys` as canonical local source; keep remote usage for drift warnings only |
| New command errors differ from existing API failure semantics | Breaks API-02 consistency | Return raw service errors through Cobra `RunE` and verify non-zero exits in command tests |

## Files Most Relevant to Phase 2 Planning

- `internal/app/cli.go`
- `internal/app/service.go`
- `internal/app/service_test.go`
- `internal/api/client.go`
- `internal/api/client_test.go`
- `internal/provider/supported.go`

## Sources

- `.planning/ROADMAP.md` (Phase 2 goal/requirements)
- `.planning/REQUIREMENTS.md` (CMD-05, CMD-06, DET-03, DET-04, API-01..03)
- `internal/app/cli.go`, `internal/app/service.go`, `internal/app/service_test.go`
- `internal/api/client.go`, `internal/api/client_test.go`
- `internal/provider/supported.go`

---

*Research completed for planning readiness.*
