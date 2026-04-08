# Phase 1 Plan 1: Deterministic Managed Block Safety

**Planned:** 2026-04-07  
**Phase:** 1 - Managed Block Safety  
**Status:** Completed (2026-04-07)

## Goal

Deliver marker-safe, idempotent `.gitignore` updates that preserve all user-owned content outside managed markers.

## Requirement Coverage

| Requirement | Planned Coverage |
|-------------|------------------|
| GIT-01 | Create `.gitignore` with managed block when missing |
| GIT-02 | Preserve existing content outside markers when no markers exist |
| GIT-03 | Replace only managed region when markers exist |
| GIT-04 | Overwrite user edits inside managed region on regeneration |
| GIT-05 | Ensure equivalent reruns are deterministic and avoid churn |

## Implementation Scope

1. Remove volatile timestamp from default managed block generation path to satisfy deterministic reruns.
2. Keep marker-bounded replacement behavior in `mergeManagedBlock` and harden malformed marker handling.
3. Preserve prepend behavior for files without markers while keeping existing user content untouched.
4. Keep dry-run behavior write-free while still reporting intended file action.

## Execution Tasks

1. Update `internal/gitignore/manager.go`:
   - Make managed block metadata deterministic by default (no `Generated at` line).
   - Add explicit malformed marker detection helper and return an error on invalid marker structure.
   - Preserve current create/prepend/replace ownership boundaries.
2. Update `internal/app/service.go`:
   - Stop passing runtime `time.Now()` into block generation.
   - Keep provider ordering and API request ordering unchanged.
3. Update tests in `internal/gitignore/manager_test.go` and `internal/app/service_test.go`:
   - Replace timestamp-coupled test setup with deterministic fixtures.
   - Add malformed marker scenario assertions.
   - Add idempotency assertion proving second equivalent run does not change file content.
4. Run project tests for touched packages and confirm green results.

## Acceptance Criteria

1. Running generation on a missing `.gitignore` creates a file with managed markers and template content.
2. Existing non-marker content remains byte-equivalent after generation.
3. Existing marker-bounded block is fully replaced, with prefix/suffix preserved.
4. Equivalent repeated runs keep `.gitignore` byte-equivalent.
5. Malformed marker structures are rejected with a clear error instead of silent rewrite.

## Risks and Mitigations

- **Risk:** Rejecting malformed markers may break existing hand-edited files.
  **Mitigation:** Return a clear, actionable error message instructing manual marker repair.
- **Risk:** Removing timestamp may reduce traceability.
  **Mitigation:** Keep provider metadata header and rely on git history for run traceability.

## Out of Scope

- Provider detection logic changes.
- API fallback/caching behavior.
- Output formatting contract changes beyond what is needed for new errors.

## Validation Commands

```bash
go test ./internal/gitignore ./internal/app
```
