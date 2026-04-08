---
phase: 02-provider-selection-api-generation
verified: 2026-04-08T09:46:48Z
status: passed
score: 5/5 must-haves verified
overrides_applied: 0
---

# Phase 2: Provider Selection & API Generation Verification Report

**Phase Goal:** Users can choose valid providers, discover available keys, and generate templates deterministically from the Toptal API.
**Verified:** 2026-04-08T09:46:48Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | User can run `gitignore-gen add <keys...>` and only missing valid providers are appended to the managed set. | ✓ VERIFIED | `internal/app/service.go:146-180` sanitizes input, preserves existing providers, appends only missing keys, and sorts `added`/`finalProviders`; covered by `TestAddAppendsOnlyMissingProviders` and `TestAddMixedSupportedUnsupportedKeys` in `internal/app/service_test.go:79-144`. |
| 2 | User can list/search supported provider keys before generating. | ✓ VERIFIED | `internal/app/cli.go:89-116` registers `list` and `search`; `internal/app/catalog.go:10-25` provides the data; spot-checks `go run . list` and `go run . search go --json` succeeded. |
| 3 | User sees unsupported keys reported as warnings while valid keys still proceed. | ✓ VERIFIED | `sanitizeKeys` in `internal/app/service.go:188-200` emits warnings and keeps valid keys; `TestAddMixedSupportedUnsupportedKeys` and `TestUnsupportedWarnings` confirm valid providers still flow through with sorted warnings. |
| 4 | User gets alphabetically ordered providers in file metadata, API requests, and command outputs. | ✓ VERIFIED | `mapKeysSorted` in `internal/app/service.go:222-228`, provider-list sorting in `internal/api/client.go:56-61`, and catalog/CLI sorting in `internal/app/catalog.go:10-25` and `internal/app/cli.go:189-197`; validated by `TestClientUsesFixtures`, `TestListProviders`, `TestSearchProviders`, `TestListCommand`, and `TestSearchCommandJSON`. |
| 5 | User gets generated template content from Toptal API, with hard failure on API errors and warning when remote list drifts from hardcoded support. | ✓ VERIFIED | `internal/api/client.go:64-87` fetches template content; `internal/app/service.go:54-60`, `92-100`, `139-170`, and `203-211` hard-fail on API errors and emit remote drift warnings; verified by `TestClientUsesFixtures`, `TestAPIFailureHardFails`, `TestTemplateAPIFailureHardFailsWithoutWrite`, `TestDetectTemplateFailureLeavesExistingGitignoreUnchanged`, and `TestRemoteProviderDriftWarning`. |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/app/service.go` | Add/detect provider sanitization, warnings, deterministic selection, API/block-write orchestration | ✓ VERIFIED | Exists, substantive (229 lines), and wired to API + gitignore manager (`AvailableProviders`, `FetchTemplate`, `BuildManagedBlock`, `UpsertManagedBlock`). |
| `internal/api/client.go` | Toptal provider-list and template API client behavior | ✓ VERIFIED | Exists, substantive (107 lines), sorts remote provider list, hard-fails non-2xx responses, returns fetched template content. |
| `internal/app/service_test.go` | Behavioral regression coverage for add/detect semantics | ✓ VERIFIED | Exists, substantive (436 lines), includes mixed-key, sorting, drift-warning, dry-run, empty-selection, and API-failure tests. |
| `internal/app/cli.go` | Cobra list/search command wiring | ✓ VERIFIED | Exists, substantive (216 lines), registers `list`/`search`, prints human and JSON discovery output, and is wired from `main.go`. |
| `internal/app/catalog.go` | Deterministic provider catalog and filtering logic | ✓ VERIFIED | Exists, substantive, sources provider data from `provider.SupportedKeys`, and sorts list/search results before returning. |
| `internal/app/cli_test.go` | CLI command behavior coverage for list/search | ✓ VERIFIED | Exists, substantive (282 lines), covers list/search human output, JSON output, and non-zero failure behavior. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `internal/app/service.go` | `internal/api/client.go` | `AvailableProviders` + `FetchTemplate` calls before block write | ✓ WIRED | Manual verification: `s.Client.AvailableProviders(ctx)` at `service.go:54,139` and `s.Client.FetchTemplate(ctx, finalProviders)` at `service.go:92,165`. `gsd-tools verify key-links` reported a false negative because the plan regex did not match field access syntax. |
| `internal/app/service.go` | `internal/gitignore/manager.go` | `BuildManagedBlock` and `UpsertManagedBlock` | ✓ WIRED | `gitignore.BuildManagedBlock(...)` at `service.go:96,169` and `s.Manager.UpsertManagedBlock(...)` at `service.go:97,170`. |
| `internal/app/cli.go` | `internal/app/catalog.go` | List/search commands call catalog functions | ✓ WIRED | `ListProviders()` at `cli.go:95` and `SearchProviders(term)` at `cli.go:110`; commands registered at `cli.go:116`. |
| `internal/app/cli.go` | `printResult` / discovery output path | Human and JSON output paths include discovery results | ✓ WIRED | `printCatalogResult(...)` at `cli.go:93-111` drives discovery output, and the CLI entrypoint is wired through `main.go:9-11`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `internal/app/service.go` | `finalProviders`, `template.Content`, warning slices | Provider keys from `sanitizeKeys`, `ReadManagedProviders`, detectors, and remote API responses via `AvailableProviders`/`FetchTemplate` | Yes | ✓ FLOWING |
| `internal/api/client.go` | `providers`, `content` | Real HTTP responses from Toptal list/template endpoints, decoded by `decodeAvailableProviders` and `io.ReadAll` | Yes | ✓ FLOWING |
| `internal/app/catalog.go` | returned `providers` slice | `provider.SupportedKeys` canonical local catalog | Yes | ✓ FLOWING |
| `internal/app/cli.go` | `CatalogResult.Providers` / rendered provider output | `ListProviders()` and `SearchProviders(term)` results, plus `CommandResult` from service methods | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Mixed valid/invalid `add` keeps valid providers and warnings | `rtk go test ./internal/app -run 'TestAddMixedSupportedUnsupportedKeys$'` | `Go test: 1 passed in 1 packages` | ✓ PASS |
| Discovery commands work and stay deterministic | `rtk go test ./internal/app -run 'Test(ListCommand|SearchCommandJSON)$'` | `Go test: 2 passed in 1 packages` | ✓ PASS |
| API failures hard-fail instead of partially succeeding | `rtk go test ./internal/app -run 'Test(APIFailureHardFails|TemplateAPIFailureHardFailsWithoutWrite)$'` | `Go test: 2 passed in 1 packages` | ✓ PASS |
| API client fixture path returns deterministic provider data | `rtk go test ./internal/api -run 'TestClientUsesFixtures$'` | `Go test: 1 passed in 1 packages` | ✓ PASS |
| Live CLI `list` command emits supported providers | `rtk go run . list` | Printed sorted provider list headed by `Command: list` / `Providers:` | ✓ PASS |
| Live CLI `search` command emits structured filtered results | `rtk go run . search go --json` | Returned JSON with sorted providers `django, go, godot, goland, ...` | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `CMD-05` | `02-01-PLAN.md` | User can run `gitignore-gen add <keys...>` to append only missing valid providers to the managed set | ✓ SATISFIED | `service.go:146-180`; `TestAddAppendsOnlyMissingProviders`, `TestAddMixedSupportedUnsupportedKeys`. |
| `CMD-06` | `02-02-PLAN.md` | User can discover available provider keys using list/search command(s) | ✓ SATISFIED | `cli.go:89-116`, `catalog.go:10-25`; `go run . list`; `go run . search go --json`; `TestListCommand`, `TestSearchCommand`, `TestSearchCommandJSON`. |
| `DET-03` | `02-01-PLAN.md` | User receives warnings for invalid/unsupported provider keys while valid keys still continue | ✓ SATISFIED | `sanitizeKeys` in `service.go:188-200`; tests confirm sorted warnings and preserved valid providers. |
| `DET-04` | `02-01-PLAN.md`, `02-02-PLAN.md` | User gets deterministic provider ordering for file metadata, API requests, and outputs | ✓ SATISFIED | Sorting in `service.go:175,222-228`, `client.go:60`, `catalog.go:12,24`, `cli.go:194-196`; tests cover API/client/catalog/CLI order. |
| `API-01` | `02-01-PLAN.md` | User gets template generation from Toptal API for the final provider set | ✓ SATISFIED | `client.go:64-87` and service fetch/write flow at `service.go:92-100` and `165-170`; `TestClientUsesFixtures`. |
| `API-02` | `02-01-PLAN.md` | User gets hard-failure behavior when API calls fail | ✓ SATISFIED | `service.go:55-57`, `93-95`, `140-142`, `166-168`; `client.go:49-50`, `79-80`; failure tests verify no write occurs. |
| `API-03` | `02-01-PLAN.md` | User receives warning if a hardcoded supported provider is missing from the remote provider list | ✓ SATISFIED | `remoteDiffWarnings` in `service.go:203-211`; `TestRemoteProviderDriftWarning`. |

### Anti-Patterns Found

No blocker, warning, or info-level anti-patterns were confirmed in the Phase 2 implementation files after scan. A broad regex matched `[]string` construction in `internal/app/cli.go:207`, but this is normal code, not a stub.

### Human Verification Required

None.

### Gaps Summary

No blocking gaps found. Phase 2 goal and all roadmap success criteria are achieved in the current codebase.

Minor note: this phase was verified after the original execution workflow, so planning metadata lagged behind implementation until the milestone closeout synced it.

---

_Verified: 2026-04-08T09:46:48Z_
_Verifier: the agent (gsd-verifier)_
