# Phase 3: Auto-Detection Workflow - Context

**Gathered:** 2026-04-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver `genignore detect` as a deterministic reset flow that infers providers from current-directory project files, runtime OS, and installed software signals, lets users refine the final set with `--include` and `--exclude`, errors when the final set is empty, and exposes detector evidence in verbose and JSON output.

</domain>

<decisions>
## Implementation Decisions

### Detection Scope
- Limit project-file detection to explicit known filenames in the current working directory root; do not add recursive traversal in v1.
- Keep detector rules as an explicit provider registry with per-provider logic so support stays deterministic and auditable.
- Combine project, OS, and installed-tool signals as a simple union of matches, then sort alphabetically.
- Treat detector-level read/parse issues as evidence for verbose/JSON output, not as command-failing errors unless the final provider set becomes empty or a later hard failure occurs.

### Selection Controls
- Reuse the existing supported-key sanitization path for `--include` and `--exclude`, including deterministic warning ordering.
- Apply selection as `detected + include - exclude`, with exclude winning last when the same key appears in both sets.
- Hard-fail immediately after final provider resolution if no providers remain, before any template fetch or file write.
- Keep unsupported include/exclude keys as warnings while still proceeding with valid selections.

### Detection Evidence
- Keep default human-readable output concise; detailed detector-by-detector evidence belongs in `--verbose` and `--json` output.
- Reuse the existing `provider.Result` contract for evidence entries instead of introducing a second detect-only payload type.
- Sort detection evidence by provider key before rendering or serializing so output stays deterministic.
- Do not show unmatched detector noise in default human-readable output.

### Command Behavior
- `detect` fully resets the managed provider set to the resolved final set; append-only behavior remains exclusive to `add`.
- `--dry-run` still runs the full detect/template/file-action flow but must not write `.gitignore`.
- Keep remote provider drift warnings visible on `detect`, just as on `add`.
- Keep phase 3 non-interactive at the CLI layer; use flags and stable output rather than prompts or wizards.

### the agent's Discretion
Implementation details inside the detector registry, evidence formatting, and internal refactors are at the agent's discretion as long as the selection semantics, determinism, and output boundaries above remain true.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/service.go` already contains the core `Detect` flow, include/exclude sanitization, remote drift warnings, and managed block update path.
- `internal/provider/detectors.go` already defines a detector registry with file, OS, PATH, and installed-application signals.
- `internal/app/types.go` already exposes `CommandResult` with `DetectionResults`, warning fields, and file action metadata.
- `internal/app/cli.go` already wires Cobra commands, shared `--json` and `--verbose` flags, and human-readable result printing.

### Established Patterns
- Sort provider selections and warnings alphabetically before writing files, calling the API, or rendering output.
- Keep unsupported keys as warnings while allowing valid work to continue.
- Use Cobra flags for behavior switches and expose the same underlying result object to human-readable and JSON output paths.
- Keep API/template failures as hard failures rather than silent fallbacks.

### Integration Points
- Extend `internal/app/cli.go`'s `detect` command contract rather than introducing a new command surface.
- Evolve `internal/app/service.go` so phase 3 behavior stays centralized with existing add/detect orchestration.
- Add or refine detectors in `internal/provider/detectors.go` and related provider helpers.
- Add regression coverage in `internal/app/service_test.go`, `internal/app/cli_test.go`, and provider-focused tests for ordering and evidence behavior.

</code_context>

<specifics>
## Specific Ideas

- Keep the phase aligned with current v1 scope: current directory only, deterministic ordering everywhere, and no new interactive CLI affordances.
- Make detector evidence useful for automation by keeping it structured and stable, while still keeping default terminal output concise.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope.

</deferred>
