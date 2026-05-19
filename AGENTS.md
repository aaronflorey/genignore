<!-- GSD:project-start source:PROJECT.md -->
## Project

**genignore**

`genignore` is a Go CLI that detects relevant `.gitignore` templates for the current project and environment, fetches remote rules from `github/gitignore`, and manages only a generated block inside `.gitignore`. It is built for developers who want safe, repeatable ignore generation without losing manual rules outside managed markers.

**Core Value:** Generate and maintain a deterministic, safe managed `.gitignore` block that users can run repeatedly without losing their own file content.

### Constraints

- **Tech stack**: Go + Cobra + Charmbracelet output with TOML-backed machine defaults
- **API dependency**: `github/gitignore` remains the live remote template source, but `runtime.offline = true` can reuse cached remote template bodies without a live refresh and `runtime.upstream_commit` pins the remote revision used for catalog and template fetches
- **Safety**: Only content between managed markers is CLI-owned — user lines outside markers must remain untouched
- **Determinism**: Provider ordering must be alphabetical — avoid output/file churn across runs
- **Scope**: Current directory only — no monorepo traversal or plugin loading in v1
<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->
## Technology Stack

## Recommended Stack
### Core Technologies
| Technology | Version | Purpose | Why Recommended | Confidence |
|------------|---------|---------|-----------------|------------|
| Go | 1.22 | Language/runtime/toolchain | Declared in `go.mod`; current project baseline for builds and tests. | HIGH |
| `github.com/spf13/cobra` | v1.9.1 | CLI command/flag framework | De-facto standard Go CLI framework (subcommands, help, completions, command tree); aligns with PRD stack. | HIGH |
| `github.com/pelletier/go-toml/v2` | v2.2.3 | Machine-level config decoding | Powers machine-level defaults loading from the path defined in `internal/app/config.go` (`configRelativePath`). | HIGH |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Human-readable terminal styling | Matches “Charmbracelet output” requirement without introducing a full TUI runtime. Good for concise status/warning formatting. | HIGH |
### Supporting Libraries
| Library | Version | Purpose | When to Use | Confidence |
|---------|---------|---------|-------------|------------|
| `os.ReadDir`, `os.Stat`, `path/filepath` | stdlib (Go 1.22) | Cross-platform filesystem inspection | Provider detectors scan the current directory entries and inspect files directly in `internal/provider/detectors.go`. | HIGH |
| `net/http`, `encoding/json`, `context` | stdlib (Go 1.26.1) | GitHub template client + structured decode | Default API layer. Keep explicit timeouts, transport limits, and typed response structs. | HIGH |
| `net/http/httptest` | stdlib (Go 1.26.1) | HTTP integration tests | Use for deterministic API fixture tests; avoid hitting live Toptal in CI. | HIGH |
| Not currently used (`testing/fstest`) | N/A | In-memory filesystem tests | Unit test detector logic against synthetic file trees quickly and deterministically. | HIGH |
| `github.com/rogpeppe/go-internal/testscript` | Not in go.mod | End-to-end CLI scenario tests | Use for fixture-driven command tests (`detect`, `add`, marker migration, dry-run, JSON shape). | HIGH |
| `github.com/google/go-cmp/cmp` | Not in go.mod | Stable assertions/golden comparisons | Use for comparing structured JSON output and provider sets with explicit diffing. | HIGH |
### Development Tools
| Tool | Purpose | Notes |
|------|---------|-------|
| `golangci-lint` v2.11.4 | Aggregated lint/quality gate | Enforce deterministic/style rules (error handling, import order, complexity caps) in CI. |
| `staticcheck` 2026.1 (v0.7.0) | Semantic/static bug detection | Keep enabled even with golangci-lint; catches API misuse and subtle correctness issues. |
| `goreleaser` v2.15.2 | Cross-platform release automation | Build/sign/package multi-OS binaries reproducibly from one config. |

Keep Go, GoReleaser, and other pinned release-tool updates isolated from feature work and verify them with `go test ./...`, `goreleaser check`, and `goreleaser release --snapshot --clean --skip=publish`.
## Installation
# Core runtime deps
# Test deps
# Dev tools (pin in CI/tooling docs)
## Alternatives Considered
| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Cobra v1.9.1 | urfave/cli v3.8.0 | Use urfave/cli only if you want a flatter, less hierarchical command model. For `detect`/`add` plus future subcommands, Cobra scales better. |
| Cobra + explicit TOML loader | Kong v1.15.0 | Kong is great for struct-tag driven CLIs, but Cobra with a small explicit config loader keeps the supported surface narrower and easier to reason about here. |
| stdlib `net/http` | Resty / generated API clients | Prefer only if API surface grows substantially (many endpoints/auth flows/retries). For one API + strong tests, stdlib is simpler and more controllable. |
## What NOT to Use
| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Full Bubble Tea TUI (`github.com/charmbracelet/bubbletea`) for v1 | Adds event-loop/state-machine complexity for a non-interactive CLI; hurts test simplicity. | Lipgloss + plain text/JSON output modes. |
| Heavy filesystem abstraction by default (Afero-first design) | Extra indirection for little gain in this scope; stdlib already gives `fs.FS` + `fstest`. | `io/fs`, `os.DirFS`, `testing/fstest`, plus thin local interfaces where needed. |
| Live API calls in core tests | Flaky CI, nondeterministic fixtures, and rate-limit/network failures. | `httptest.Server` + checked-in API fixtures (as PRD requires). |
| Map-based output assembly for provider sets | Go map iteration order is randomized; causes output/file churn. | Typed structs + sorted slices before serialize/write. |
## Stack Patterns by Variant
- Use Cobra plus the explicit TOML config loader only.
- Keep all runtime behavior explicit via flags and command args.
- Keep Cobra.
- Expand config loading only when a new supported input surface is intentionally added and documented.
## Version Compatibility
| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| Go 1.22 | Cobra v1.9.1 | This repository currently pins both in `go.mod`. |
| Go 1.26.1 | Viper v1.21.0 | Viper README indicates Go >=1.23; compatible with 1.26.x. |
| Go 1.26.1 | Staticcheck 2026.1 | Release notes explicitly mention Go 1.26 support updates. |
## Prescriptive Implementation Notes
## Sources
- https://go.dev/VERSION?m=text — verified latest stable Go (`go1.26.1`)
- https://pkg.go.dev/path/filepath#WalkDir — deterministic lexical walk behavior
- https://pkg.go.dev/io/fs — FS abstractions and testing guidance
- https://pkg.go.dev/net/http
- https://pkg.go.dev/net/http/httptest
- https://pkg.go.dev/testing/fstest
- https://pkg.go.dev/github.com/spf13/cobra
- https://api.github.com/repos/spf13/cobra/releases/latest
- https://pkg.go.dev/github.com/pelletier/go-toml/v2
- https://api.github.com/repos/charmbracelet/lipgloss/releases/latest
- https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript
- https://api.github.com/repos/rogpeppe/go-internal/releases/latest
- https://api.github.com/repos/google/go-cmp/releases/latest
- https://api.github.com/repos/golangci/golangci-lint/releases/latest
- https://api.github.com/repos/dominikh/go-tools/releases/latest
- https://api.github.com/repos/goreleaser/goreleaser/releases/latest
- https://github.com/github/gitignore
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

- Keep fixture repositories under `testdata/repos/` reduced to detector-relevant files only; strip secrets, history, vendored code, and unrelated source.
- Keep machine-readable stability contracts under `testdata/contracts/` and update them only with intentional, reviewed output changes.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, or `.github/skills/` with an index file in each skill directory.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
