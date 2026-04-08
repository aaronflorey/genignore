# Phase 4: Output & Automation Contracts - Context

**Gathered:** 2026-04-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver stable human-readable and machine-readable command outputs for `detect`, `add`, `list`, and `search`, including a trustworthy dry-run contract, deterministic warnings and evidence ordering, and JSON payloads that automation can consume safely.

</domain>

<decisions>
## Implementation Decisions

### Human Output
- Keep default terminal output as fixed labeled summary lines rather than prose or table-heavy output.
- Show only populated summary sections by default so successful runs stay concise but still expose detected or selected providers, warnings, and file action.
- Append detailed detector evidence only when `--verbose` is set, after the concise summary.
- Keep `list` and `search` aligned with the same compact command/query plus provider-line style.

### JSON Contract
- Keep a stable JSON contract with shared top-level metadata and command-specific optional fields rather than inventing a different ad hoc shape for each command.
- Preserve existing camelCase field naming and treat command metadata, final selections, warnings, and file action as contract-stable fields.
- Include sorted structured `detectionResults` when detection runs, reusing the existing provider result shape.
- Derive human and JSON outputs from the same result structs to minimize drift.

### Dry-Run Contract
- Keep dry-run as a normal success path with an explicit dry-run file action in both human and JSON output.
- Run the same selection, warning, template, and file-action logic as a live command, stopping only before file write.
- Do not add full `.gitignore` content previews or diffs in phase 4; keep this phase focused on stable summaries and metadata.
- Dry-run should exit successfully whenever the equivalent live command would succeed.

### Warning And Error Boundaries
- Keep warnings inside successful human and JSON output, while true execution failures still go to stderr with a non-zero exit.
- Unsupported keys and remote provider drift remain warnings when valid work can continue.
- Empty detect results, API failures, and malformed marker or write failures remain hard stops with no partial writes.
- Keep warning arrays and structured output deterministically ordered before rendering or serialization.

### the agent's Discretion
The agent has discretion over renderer refactors, shared helper extraction, and exact summary wording so long as the output contract, determinism, and warning/error boundaries above remain stable.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/cli.go` already owns all human-readable and JSON rendering paths for command results and catalog results.
- `internal/app/types.go` already defines the primary machine-readable result contract for `detect` and `add`.
- `internal/app/json_test.go` and `internal/app/cli_test.go` already provide a base for JSON- and stdout-level contract tests.
- `internal/app/service.go` already computes deterministic selections, warnings, detection evidence, and file actions that can feed more formal output contracts.

### Established Patterns
- Keep outputs derived from typed structs rather than map assembly.
- Sort user-facing collections before printing or serializing.
- Keep terminal output compact and labeled.
- Treat dry-run as a write-free execution of the real command path, not a separate fake mode.

### Integration Points
- Extend `internal/app/cli.go` renderers and flag handling rather than creating a second output layer.
- Evolve `internal/app/types.go` if phase 4 needs stronger shared payload shapes across commands.
- Add contract-focused tests in `internal/app/cli_test.go`, `internal/app/json_test.go`, and service-level tests for dry-run and warning semantics.

</code_context>

<specifics>
## Specific Ideas

- Preserve the compact terminal UX established so far while making the payloads reliable enough for scripts and CI.
- Use phase 4 to lock ordering and presence rules that phase 5 can then guard with broader regression coverage.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope.

</deferred>
