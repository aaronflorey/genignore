# AGENTS.md

## Project Overview

`genignore` is a Go 1.22 Cobra CLI. It detects providers for the current directory, loads embedded `.gitignore` template content from the checked-in `github/gitignore` snapshot plus local custom templates, and updates only the managed marker block in `.gitignore`.

## Source Of Truth

- Module and Go version: `go.mod`, `go.sum`
- CLI entrypoint: `main.go` -> `internal/app/cli.go` -> `internal/app/service.go`
- Managed block behavior: `internal/gitignore/manager.go`
- Provider detection: `internal/provider/`, `internal/rulecatalog/rules.json`
- Embedded upstream templates: git submodule `internal/templatecatalog/github-gitignore` from `.gitmodules`
- Local custom templates: `internal/customtemplate/definitions.go` and `internal/customtemplate/templates/`
- CI and release automation: `.github/workflows/ci.yml`, `.github/workflows/release-please.yml`, `.goreleaser.yaml`, `release-please-config.json`
- Detailed docs: `docs/DEVELOPMENT.md`, `docs/TESTING.md`, `docs/ARCHITECTURE.md`, `docs/CONFIGURATION.md`

## Commands

| Purpose | Command | When to run |
| --- | --- | --- |
| Download deps | `go mod download` | Fresh checkout or dependency changes |
| Format Go | `go fmt ./...` | After Go edits |
| Build | `go build ./...` | After CLI or package changes |
| Test all | `go test ./...` | Before handoff when practical; this also validates embedded catalog assets |
| Test one package | `go test ./internal/provider` | Focused package iteration |
| Test one case | `go test ./internal/app -run TestListCommand` | Focused test iteration |
| Coverage | `go test ./... -coverprofile=coverage.out` | When coverage output is needed |
| Run CLI from source | `go run . detect --dry-run` | Manual smoke test without writing `.gitignore` |
| Validate GoReleaser | `mise x -- goreleaser check` | Release config or packaging changes |
| Snapshot release build | `mise x -- goreleaser release --snapshot --clean --skip=publish` | Release-impacting changes |

## Development Notes

- There is no npm-style task runner; use Go commands directly plus `mise x -- goreleaser ...` for release tooling.
- CI checks out submodules recursively. If template snapshot tests fail locally, confirm `internal/templatecatalog/github-gitignore` is initialized and in sync.
- Normal CLI execution should not require network access; provider catalogs, template bodies, and repository detection rules are embedded.
- Machine config is optional and strict TOML at `$HOME/.config/genignore/config.toml`; only `[defaults]`, `providers`, and `ignore_rules` are supported.
- `internal/api` is a legacy package name; current behavior is embedded catalog/template loading, not live API fetching.

## Testing And Fixtures

- Tests use Go's standard `testing` package and live next to implementation as `*_test.go`.
- Reuse CLI output helpers in `internal/app/cli_test.go` for command tests.
- Keep `testdata/repos/` fixtures minimal, detector-relevant, and secret-free.
- `testdata/contracts/` contains machine-readable output contracts; update only for intentional output changes, then review provider ordering, provenance, and formatting churn.
- For intentional fixture-backed contract changes, run `go test ./internal/provider ./internal/gitignore ./internal/app` before accepting the diff.
- Managed-block benchmarks are intentional only: `go test -run '^$' -bench . ./internal/gitignore`.
- Fuzz targets are intentional only: `go test -run '^$' -fuzz=FuzzParseManagedProvidersRoundTrip -fuzztime=10s ./internal/gitignore` and `go test -run '^$' -fuzz=FuzzMergeManagedBlock -fuzztime=10s ./internal/gitignore`.

## Invariants To Preserve

- Only content between `# BEGIN genignore` and `# END genignore` is CLI-owned; preserve user content outside markers.
- Provider ordering and generated output must stay deterministic and alphabetically stable.
- Scope is the current directory only; do not add monorepo traversal or plugin loading without an explicit product change.
- Managed blocks must include deterministic provenance for upstream and embedded custom providers.
- Required env rules are normalized by `internal/gitignore/manager.go`: `.env`, `.env.*`, `!.env.example`, `!.env.ci`.

## Catalog And Template Changes

- To add a custom provider, add `internal/customtemplate/templates/<key>.gitignore`, register it in `internal/customtemplate/definitions.go`, and add/update tests.
- To change repository-backed detection, edit `internal/rulecatalog/rules.json` and keep schema/tests passing in `internal/rulecatalog` and `internal/provider`.
- Treat upstream `github/gitignore` snapshot refreshes as narrow maintenance changes because they can churn provider catalogs and generated contract output.

## Release Notes

- Release PRs are managed by release-please on `main`; conventional commit types in `release-please-config.json` affect changelog sections.
- Packaging uses GoReleaser v2.15.2 and Homebrew publishing expects `HOMEBREW_TAP_GITHUB_TOKEN` only in release CI; never add real tokens to the repo.
- For release artifact/offline behavior checks, follow `docs/TESTING.md#manual-offline-release-verification`.
