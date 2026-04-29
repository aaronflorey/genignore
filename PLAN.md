# genignore Plan

This plan replaces the older PRD and focuses on the next actionable improvements based on the current codebase.

## Current Findings

- The CLI already implements `detect`, `add`, `list`, and `search`.
- The implementation is split across `internal/app`, `internal/gitignore`, `internal/provider`, and `internal/api`.
- The codebase already has targeted tests for API, provider detection, JSON output, CLI behavior, and managed block updates.
- The removed `PRD.md` described a narrower command and scope model than the implementation now exposes.

## Priorities

### P0: Document The Actual CLI Surface

- Keep planning and user docs aligned with the implemented commands: `detect`, `add`, `list`, `search`.
- Document the supported output modes and safety guarantees that already exist in code:
  `--dry-run`, `--json`, deterministic provider ordering, and managed-block-only ownership.
- Remove or avoid product docs that describe commands or constraints the binary does not match.

### P1: Capture Architecture And Invariants

- Add a short architecture note for the current package split:
  `internal/app` for command orchestration, `internal/provider` for detection, `internal/api` for Toptal access, and `internal/gitignore` for managed block updates.
- Record the invariants that should not regress:
  deterministic provider ordering, API-required runtime behavior, overwrite-only inside managed markers, and preservation of user content outside the block.

### P1: Tighten Detection And Output Documentation

- Document the current detection model as implemented, including project signals, OS/tool detection, unsupported-key warnings, and verbose or JSON reporting.
- Keep examples grounded in behavior already covered by tests instead of aspirational product language.

### P2: Turn Existing Coverage Into An Explicit Validation Checklist

- Maintain a lightweight checklist for the behaviors already exercised by tests:
  managed block creation and replacement, `detect` reset behavior, `add` append-only behavior, ordering stability, unsupported-key handling, API failures, and JSON shape.
- Use that checklist when changing scope or command behavior so docs and tests move together.
