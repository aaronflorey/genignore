# Stack Research

**Domain:** Cross-platform Go CLI (filesystem scanning + HTTP API + deterministic output + strong testability)
**Researched:** 2026-04-07
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended | Confidence |
|------------|---------|---------|-----------------|------------|
| Go | 1.26.1 | Language/runtime/toolchain | Current stable release; standard library already covers file walking, HTTP, JSON, and test infrastructure with low dependency risk. | HIGH |
| `github.com/spf13/cobra` | v1.10.2 | CLI command/flag framework | De-facto standard Go CLI framework (subcommands, help, completions, command tree); aligns with PRD stack. | HIGH |
| `github.com/spf13/viper` | v1.21.0 | Flag/env config binding | Use narrowly for flag+env binding and precedence, not full config-file complexity in v1. Keeps CLI ergonomics while respecting scope (no config file). | HIGH |
| `github.com/charmbracelet/lipgloss` | v2.0.2 | Human-readable terminal styling | Matches “Charmbracelet output” requirement without introducing a full TUI runtime. Good for concise status/warning formatting. | HIGH |

### Supporting Libraries

| Library | Version | Purpose | When to Use | Confidence |
|---------|---------|---------|-------------|------------|
| `io/fs`, `path/filepath` (`WalkDir`) | stdlib (Go 1.26.1) | Cross-platform filesystem traversal | Primary detector scan path. `WalkDir` is deterministic (lexical order) and faster than legacy `Walk`. | HIGH |
| `net/http`, `encoding/json`, `context` | stdlib (Go 1.26.1) | Toptal API client + structured decode | Default API layer. Keep explicit timeouts, transport limits, and typed response structs. | HIGH |
| `net/http/httptest` | stdlib (Go 1.26.1) | HTTP integration tests | Use for deterministic API fixture tests; avoid hitting live Toptal in CI. | HIGH |
| `testing/fstest` | stdlib (Go 1.26.1) | In-memory filesystem tests | Unit test detector logic against synthetic file trees quickly and deterministically. | HIGH |
| `github.com/rogpeppe/go-internal/testscript` | v1.14.1 | End-to-end CLI scenario tests | Use for fixture-driven command tests (`detect`, `add`, marker migration, dry-run, JSON shape). | HIGH |
| `github.com/google/go-cmp/cmp` | v0.7.0 | Stable assertions/golden comparisons | Use for comparing structured JSON output and provider sets with explicit diffing. | HIGH |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `golangci-lint` v2.11.4 | Aggregated lint/quality gate | Enforce deterministic/style rules (error handling, import order, complexity caps) in CI. |
| `staticcheck` 2026.1 (v0.7.0) | Semantic/static bug detection | Keep enabled even with golangci-lint; catches API misuse and subtle correctness issues. |
| `goreleaser` v2.15.2 | Cross-platform release automation | Build/sign/package multi-OS binaries reproducibly from one config. |

## Installation

```bash
# Core runtime deps
go get github.com/spf13/cobra@v1.10.2
go get github.com/spf13/viper@v1.21.0
go get github.com/charmbracelet/lipgloss@v2.0.2

# Test deps
go get github.com/rogpeppe/go-internal/testscript@v1.14.1
go get github.com/google/go-cmp/cmp@v0.7.0

# Dev tools (pin in CI/tooling docs)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.11.4
go install honnef.co/go/tools/cmd/staticcheck@2026.1
go install github.com/goreleaser/goreleaser/v2@v2.15.2
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Cobra v1.10.2 | urfave/cli v3.8.0 | Use urfave/cli only if you want a flatter, less hierarchical command model. For `detect`/`add` plus future subcommands, Cobra scales better. |
| Cobra+Viper | Kong v1.15.0 | Kong is great for struct-tag driven CLIs, but Cobra+Viper is more ecosystem-standard for long-lived multi-command tools and shell completion workflows. |
| stdlib `net/http` | Resty / generated API clients | Prefer only if API surface grows substantially (many endpoints/auth flows/retries). For one API + strong tests, stdlib is simpler and more controllable. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Full Bubble Tea TUI (`github.com/charmbracelet/bubbletea`) for v1 | Adds event-loop/state-machine complexity for a non-interactive CLI; hurts test simplicity. | Lipgloss + plain text/JSON output modes. |
| Heavy filesystem abstraction by default (Afero-first design) | Extra indirection for little gain in this scope; stdlib already gives `fs.FS` + `fstest`. | `io/fs`, `os.DirFS`, `testing/fstest`, plus thin local interfaces where needed. |
| Live API calls in core tests | Flaky CI, nondeterministic fixtures, and rate-limit/network failures. | `httptest.Server` + checked-in API fixtures (as PRD requires). |
| Map-based output assembly for provider sets | Go map iteration order is randomized; causes output/file churn. | Typed structs + sorted slices before serialize/write. |

## Stack Patterns by Variant

**If scope stays v1 (no persistent config file):**
- Use Cobra + limited Viper (`BindPFlags`, `AutomaticEnv`) only.
- Keep all runtime behavior explicit via flags and command args.

**If v2 adds user config files:**
- Keep Cobra.
- Expand Viper to config-file loading with strict schema unmarshal + validation layer (do not access Viper globally throughout code).

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| Go 1.26.1 | Cobra v1.10.2 | Cobra release metadata indicates current support/tooling updates through recent Go versions. |
| Go 1.26.1 | Viper v1.21.0 | Viper README indicates Go >=1.23; compatible with 1.26.x. |
| Go 1.26.1 | Staticcheck 2026.1 | Release notes explicitly mention Go 1.26 support updates. |

## Prescriptive Implementation Notes

1. **Use stdlib first for core behavior** (scan, HTTP, JSON, sorting). This is the strongest path to deterministic output and low-maintenance tests.
2. **Treat Viper as a boundary adapter**, not global state. Read once, map into typed options struct, pass that struct downward.
3. **Design for determinism by default**: sort provider keys before storage, API calls, and output rendering.
4. **Test pyramid for this CLI**:
   - detector unit tests (`fstest.MapFS`)
   - API/client tests (`httptest` + fixture payloads)
   - E2E command tests (`testscript` with fixture `.gitignore` scenarios)

## Sources

- https://go.dev/VERSION?m=text — verified latest stable Go (`go1.26.1`)
- https://pkg.go.dev/path/filepath#WalkDir — deterministic lexical walk behavior
- https://pkg.go.dev/io/fs — FS abstractions and testing guidance
- https://pkg.go.dev/net/http
- https://pkg.go.dev/net/http/httptest
- https://pkg.go.dev/testing/fstest
- https://pkg.go.dev/github.com/spf13/cobra
- https://api.github.com/repos/spf13/cobra/releases/latest
- https://pkg.go.dev/github.com/spf13/viper
- https://api.github.com/repos/spf13/viper/releases/latest
- https://api.github.com/repos/charmbracelet/lipgloss/releases/latest
- https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript
- https://api.github.com/repos/rogpeppe/go-internal/releases/latest
- https://api.github.com/repos/google/go-cmp/releases/latest
- https://api.github.com/repos/golangci/golangci-lint/releases/latest
- https://api.github.com/repos/dominikh/go-tools/releases/latest
- https://api.github.com/repos/goreleaser/goreleaser/releases/latest
- https://www.toptal.com/developers/gitignore/api/node,go

---
*Stack research for: gitignore generation CLI*
*Researched: 2026-04-07*
