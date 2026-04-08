# Phase 1: Managed Block Safety - Research

**Researched:** 2026-04-07  
**Domain:** Safe, deterministic `.gitignore` managed-region mutation in Go CLI  
**Confidence:** HIGH

## User Constraints

No `*-CONTEXT.md` exists for this phase, so there are no additional locked user decisions beyond existing project requirements/state docs. [VERIFIED: `.planning/STATE.md` line 14 + user prompt]

### Locked Decisions
- None provided in phase context file. [VERIFIED: user prompt "No phase context file currently exists"]

### the agent's Discretion
- Full implementation approach for marker-safe file mutation, as long as PRD/requirements constraints are honored. [VERIFIED: `.planning/REQUIREMENTS.md` + `.planning/ROADMAP.md`]

### Deferred Ideas (OUT OF SCOPE)
- Monorepo mode, plugin system, offline/local fallback, persistent cache, and project config file are out of scope for v1. [VERIFIED: `.planning/REQUIREMENTS.md` lines 59-72]

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GIT-01 | Create `.gitignore` with managed block if missing | Upsert/create path, file action modeling, and atomic-write recommendation |
| GIT-02 | Preserve all existing content when no markers | Prepend managed block without mutating existing lines |
| GIT-03 | Replace only managed region when markers exist | Marker-bounded splice algorithm with explicit malformed-marker handling |
| GIT-04 | Overwrite user edits inside managed region | Managed region is always regenerated from current provider/template set |
| GIT-05 | Idempotent repeated runs | Deterministic ordering and removal/gating of volatile metadata in managed block |

## Summary

Phase 1 is primarily a **file mutation correctness** phase, not a provider-detection phase. The safest implementation is a bounded marker splice strategy: parse existing `.gitignore` into `prefix`, `managed`, and `suffix`, then rewrite only `managed` (or prepend managed block when no markers exist). This directly maps to GIT-01 through GIT-04. [VERIFIED: `.planning/ROADMAP.md` lines 24-31, `.planning/REQUIREMENTS.md` lines 28-32]

Current project code already has the right high-level shape (`internal/gitignore/manager.go`), but there is a deterministic-risk detail: service paths currently pass `time.Now()` into block generation, which can violate strict idempotency/file-stability expectations unless timestamp behavior is explicitly controlled. [VERIFIED: `internal/app/service.go` lines 93 and 145, `internal/gitignore/manager.go` lines 32-39]

**Primary recommendation:** Keep marker-bounded mutation in `internal/gitignore.Manager`, and make block content deterministic by default (no volatile timestamp in default managed payload). [VERIFIED: requirements + codebase]

## Project Constraints (from AGENTS.md)

- Use Go + Cobra/Viper + Charmbracelet output stack. [VERIFIED: `AGENTS.md` lines 12-13]
- Toptal gitignore API is required at runtime; no local fallback in v1. [VERIFIED: `AGENTS.md` line 13]
- CLI owns only content between managed markers; external lines must remain untouched. [VERIFIED: `AGENTS.md` line 14]
- Deterministic alphabetical ordering is required. [VERIFIED: `AGENTS.md` line 15]
- Scope is current directory only (no monorepo/plugin loading in v1). [VERIFIED: `AGENTS.md` line 16]

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go toolchain | 1.26.1 | Language/runtime | Current stable toolchain; already installed in environment. [VERIFIED: go.dev VERSION + local `go version`] |
| `os` (stdlib) | go1.26.1 stdlib | Read/write `.gitignore` | Official file API; `WriteFile` create/truncate semantics are explicit. [CITED: https://pkg.go.dev/os#WriteFile + VERIFIED: `go doc os.WriteFile`] |
| `strings` (stdlib) | go1.26.1 stdlib | Marker parsing/splicing | `Cut`/`Index` provide simple bounded split logic for markers. [CITED: https://pkg.go.dev/strings#Cut + VERIFIED: `go doc strings.Cut`] |
| `sort` (stdlib) | go1.26.1 stdlib | Deterministic provider ordering | `sort.Strings` guarantees increasing order. [VERIFIED: `go doc sort.Strings`] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/spf13/cobra` | v1.10.2 latest | CLI surface | Keep existing command plumbing; not phase-critical but project-standard. [VERIFIED: https://proxy.golang.org/github.com/spf13/cobra/@latest] |
| `github.com/spf13/viper` | v1.21.0 latest | flag/env wiring | Use only for config binding; not for managed-block mutation logic. [VERIFIED: https://proxy.golang.org/github.com/spf13/viper/@latest] |
| `github.com/charmbracelet/lipgloss` | v1.1.0 latest | human output styling | Optional for human output; not required for mutation correctness. [VERIFIED: https://proxy.golang.org/github.com/charmbracelet/lipgloss/@latest] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| String marker splice (`strings`) | Regex-based block replacement | Regex is less explicit for malformed/duplicate marker handling and harder to reason about for edge cases. [ASSUMED] |
| Direct `os.WriteFile` | temp-file + `Sync` + `Rename` | Safer against partial writes; slightly more code and platform nuances. [CITED: https://pkg.go.dev/os#WriteFile, https://pkg.go.dev/os#Rename + VERIFIED: `go doc os.File.Sync`] |

**Version verification notes:**
- Repo currently pins Cobra `v1.9.1`, Viper `v1.20.1`, Lipgloss `v1.1.0`. [VERIFIED: `go.mod` lines 6-8]
- Latest available now: Cobra `v1.10.2`, Viper `v1.21.0`, Lipgloss `v1.1.0`. [VERIFIED: Go module proxy `@latest` endpoints]

## Architecture Patterns

### Recommended Project Structure
```text
internal/
├── gitignore/
│   ├── manager.go        # marker parsing + safe upsert logic
│   └── manager_test.go   # migration/idempotency edge-case tests
├── app/
│   └── service.go        # orchestration (providers/api -> block -> manager)
└── api/
    └── client.go         # template/provider API calls
```

### Pattern 1: Three-way splice (prefix/managed/suffix)
**What:** Parse existing file once, identify marker bounds, and rewrite only bounded segment. [VERIFIED: `internal/gitignore/manager.go`]
**When to use:** Every write path for GIT-01..GIT-04.
**Example:**
```go
// Source: internal/gitignore/manager.go + https://pkg.go.dev/strings#Index
start := strings.Index(existing, StartMarker)
end := strings.Index(existing, EndMarker)
if start == -1 || end == -1 || end < start {
    // prepend block + preserve existing content
}
end += len(EndMarker)
if end < len(existing) && existing[end] == '\n' {
    end++
}
updated := existing[:start] + block + existing[end:]
```

### Pattern 2: Determinism-first block generation
**What:** Inputs to block generation must be stable for equivalent runs (sorted providers, stable template payload, no volatile fields by default). [VERIFIED: requirements + codebase]
**When to use:** Any code path producing managed block text.
**Example:**
```go
// Source: internal/app/service.go + go doc sort.Strings
finalProviders := mapKeysSorted(final) // sorted before write
// Avoid time.Now() in default deterministic write path.
```

### Anti-Patterns to Avoid
- **Whole-file overwrite without marker bounds:** risks data loss outside managed ownership. [VERIFIED: PRD managed-block behavior]
- **Volatile metadata in default managed payload (e.g., current timestamp):** creates non-functional churn across equivalent runs. [VERIFIED: GIT-05 + current `time.Now()` usage]
- **Silent malformed-marker behavior:** if marker pairing is broken, blindly proceeding can duplicate blocks or corrupt intent. [ASSUMED]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File write reliability | Custom buffered writer with ad-hoc crash handling | `os.CreateTemp` + write + `File.Sync` + `os.Rename` | `WriteFile` can leave partial state on mid-failure; stdlib already provides primitives. [VERIFIED: `go doc os.WriteFile`, `go doc os.CreateTemp`, `go doc os.File.Sync`, `go doc os.Rename`] |
| `.gitignore` pattern semantics | Custom parser/normalizer for template rules | Treat API template as opaque text; only manage marker ownership | Gitignore syntax is rich (`!`, `**`, escaping, precedence). Re-parsing increases risk. [CITED: https://git-scm.com/docs/gitignore] |
| Deterministic ordering | Manual bespoke sort logic | `sort.Strings` / `slices.Sort` | Standard library guarantees increasing order and is clearer. [VERIFIED: `go doc sort.Strings`] |

**Key insight:** This phase should hand-roll only **marker ownership boundaries**, not gitignore semantics or complex write durability primitives already covered by stdlib. [VERIFIED + CITED]

## Common Pitfalls

### Pitfall 1: Timestamp-induced churn
**What goes wrong:** Equivalent runs still modify managed block due to regenerated timestamp. [VERIFIED: current code passes `time.Now()`]
**Why it happens:** Timestamp is embedded in content, so textual output changes each run.
**How to avoid:** Remove timestamp from default managed block, or gate behind non-default verbose/debug mode. [ASSUMED]
**Warning signs:** `git diff` shows only `# Generated at:` changed between runs. [VERIFIED: `manager.go` block format]

### Pitfall 2: Non-atomic writes on interruption
**What goes wrong:** `.gitignore` can be partially written if write fails mid-operation. [VERIFIED: `go doc os.WriteFile`]
**Why it happens:** `WriteFile` uses multiple syscalls; docs explicitly warn about partial state.
**How to avoid:** Temp-file write + sync + rename in same directory. [CITED: https://pkg.go.dev/os#WriteFile, https://pkg.go.dev/os#Rename]
**Warning signs:** Truncated/malformed `.gitignore` after process crash or I/O fault.

### Pitfall 3: Ambiguous marker handling
**What goes wrong:** Duplicate or mismatched markers can lead to unexpected replacement boundaries. [ASSUMED]
**Why it happens:** naive first-start/first-end scanning may not represent intended block when file is malformed. [VERIFIED: current merge logic uses first `Index`]
**How to avoid:** Validate marker count/order; hard-fail on invalid structure and require manual repair. [ASSUMED]
**Warning signs:** Multiple BEGIN/END markers or END preceding BEGIN.

## Code Examples

Verified patterns from official/project sources:

### Create-if-missing / upsert existing
```go
// Source: internal/gitignore/manager.go
content, err := os.ReadFile(path)
if err != nil && !os.IsNotExist(err) {
    return err
}
if os.IsNotExist(err) {
    return os.WriteFile(path, []byte(block), 0o644)
}
updated := mergeManagedBlock(string(content), block)
return os.WriteFile(path, []byte(updated), 0o644)
```

### Safer durable write pattern
```go
// Source: os docs (WriteFile/CreateTemp/Sync/Rename)
tmp, err := os.CreateTemp(filepath.Dir(path), ".genignore-*")
if err != nil { return err }
defer os.Remove(tmp.Name())

if _, err := tmp.Write([]byte(updated)); err != nil { return err }
if err := tmp.Sync(); err != nil { return err }
if err := tmp.Close(); err != nil { return err }
if err := os.Rename(tmp.Name(), path); err != nil { return err }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Treat `os.WriteFile` as "good enough" for all updates | Account for partial-write caveat in critical file mutation paths | Documented in current stdlib docs | Prefer temp-write+rename for stronger safety in ownership files. [VERIFIED: `go doc os.WriteFile`, `go doc os.Rename`] |
| Assume planned stack versions from prior research remain current | Re-verify package/tool versions against registry before planning | Ongoing best practice | Prevents stale guidance (example: Cobra/Viper advanced beyond current repo pins). [VERIFIED: go.mod + module proxy] |

**Deprecated/outdated:**
- Prior stack note claiming Lipgloss `v2.0.2` is current appears outdated; module proxy reports latest `v1.1.0`. [VERIFIED: `AGENTS.md` vs `https://proxy.golang.org/github.com/charmbracelet/lipgloss/@latest`]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Regex replacement is materially worse than string-boundary splice for this use-case | Standard Stack / Alternatives | Could over-constrain implementation style unnecessarily |
| A2 | Default mode should hide/remove timestamp metadata for idempotency UX | Architecture Patterns / Pitfalls | Could conflict with explicit product requirement for visible timestamp |
| A3 | Invalid marker structure should hard-fail rather than auto-heal | Common Pitfalls | Could reduce convenience if product prefers auto-repair |

## Open Questions

1. **Should timestamp remain in managed block by default?**
   - What we know: GIT-05 requires deterministic repeated runs, and current implementation injects `time.Now()`. [VERIFIED]
   - What's unclear: Product preference for traceability vs zero-churn defaults.
   - Recommendation: Decide in planning as explicit acceptance criterion for idempotency semantics.

2. **How should malformed/duplicate markers be handled?**
   - What we know: Current merge logic relies on first BEGIN/first END indexes. [VERIFIED]
   - What's unclear: Desired behavior (hard error vs best-effort rewrite).
   - Recommendation: Lock one policy and encode with tests before implementation tasks.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Build/test/implementation | ✓ | go1.26.1 | — |
| Git | Manual validation (`git diff` idempotency checks) | ✓ | 2.50.1 | — |

**Missing dependencies with no fallback:**
- None. [VERIFIED: local commands]

**Missing dependencies with fallback:**
- None.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A (local CLI file mutation) [VERIFIED: phase scope docs] |
| V3 Session Management | no | N/A [VERIFIED: phase scope docs] |
| V4 Access Control | no | N/A [VERIFIED: phase scope docs] |
| V5 Input Validation | yes | Validate markers/structure and supported provider keys before write path decisions. [VERIFIED: `sanitizeKeys` + marker logic] |
| V6 Cryptography | no | N/A for this phase [VERIFIED: requirements scope] |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Partial write corrupts `.gitignore` | Tampering | temp-write + `Sync` + `Rename`; fail closed on write errors. [VERIFIED/CITED] |
| Marker spoofing/duplication causes incorrect replacement range | Tampering | marker count/order validation and explicit malformed-file error. [ASSUMED] |
| Untrusted template content changes ignore semantics unexpectedly | Tampering | treat source as authoritative API output for v1 and surface provider provenance in metadata/logging. [VERIFIED: PRD API behavior + ASSUMED mitigation details] |

## Sources

### Primary (HIGH confidence)
- `.planning/REQUIREMENTS.md` (GIT-01..GIT-05 definitions and scope)
- `.planning/ROADMAP.md` (Phase 1 goal/success criteria)
- `AGENTS.md` (project constraints/stack)
- `internal/gitignore/manager.go`, `internal/app/service.go`, `go.mod` (current implementation state)
- `go doc os.WriteFile`, `go doc os.CreateTemp`, `go doc os.File.Sync`, `go doc os.Rename`, `go doc strings.Cut`, `go doc sort.Strings`
- https://proxy.golang.org/github.com/spf13/cobra/@latest
- https://proxy.golang.org/github.com/spf13/viper/@latest
- https://proxy.golang.org/github.com/charmbracelet/lipgloss/@latest
- https://go.dev/VERSION?m=text
- https://git-scm.com/docs/gitignore

### Secondary (MEDIUM confidence)
- None.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** — based on current module proxy and local/go docs.
- Architecture: **HIGH** — based on explicit phase requirements + existing code paths.
- Pitfalls: **MEDIUM** — strongest for timestamp/partial-write; malformed-marker policy still needs product decision.

**Research date:** 2026-04-07  
**Valid until:** 2026-05-07
