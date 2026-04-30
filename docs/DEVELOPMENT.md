<!-- generated-by: gsd-doc-writer -->
# Development

## Local setup

1. Fork the repository, then clone your fork and enter the project directory:

```bash
git clone <your-fork-url>
cd genignore
```

2. Confirm your toolchain matches module requirements (`go 1.22` in `go.mod`):

```bash
go version
```

3. Sync module dependencies:

```bash
go mod tidy
```

4. Build and test before opening a PR:

```bash
go build ./...
go test ./...
```

5. Run commands directly from source while developing:

```bash
go run . detect --dry-run
```

## Build commands

This repository does not use `package.json` scripts. Development and validation use Go module commands plus release tooling.

| Command | Description |
| --- | --- |
| `go build ./...` | Compile all packages in this module. |
| `go test ./...` | Run all tests across `internal/...` packages. |
| `go run . detect` | Run provider detection from source and update the managed `.gitignore` block. |
| `go run . detect --dry-run` | Preview detection output and file action without writing. |
| `go run . add <keys...>` | Add provider keys to the existing managed set. |
| `go run . list` | Print all supported provider keys from the provider catalog. |
| `go run . search <term>` | Search provider keys by term. |
| `goreleaser check` | Validate `.goreleaser.yaml` locally (equivalent intent to CI release validation). |
| `goreleaser build --snapshot --clean` | Build release artifacts locally without publishing a release. |

## Code style

- **Formatting:** use standard Go formatting (`gofmt`/`go fmt`) for changed files.
- **Linting tool:** CI runs `golangci/golangci-lint-action@v8` in `.github/workflows/ci.yml` (`lint-and-test` job, `Lint` step).
- **Lint configuration:** no repo-local `.golangci.yml`/`.golangci.yaml`/`.golangci.toml` is present, so workflow defaults and action configuration are the active lint baseline.
- **CI quality gate:** the same CI job also runs `go test ./...`.

## Branch conventions

- The repository’s release workflow runs on pushes to `main` (`.github/workflows/release-please.yml`), so `main` is the effective default branch.
- No explicit branch naming pattern is documented in `CONTRIBUTING.md` or `.github/PULL_REQUEST_TEMPLATE.md` (both are absent).

## PR process

- Open your PR against `main`.
- Ensure GitHub Actions checks pass in `.github/workflows/ci.yml` (`lint-and-test` and `release-validation`).
- Use conventional commit types (`feat`, `fix`, `perf`, `docs`, `test`, `refactor`, `chore`) so `release-please` can map commit types to changelog sections (`release-please-config.json`).
- Keep PR scope focused and include or update tests when behavior changes.
- If changes affect packaging, validate release configuration with `goreleaser check` and confirm snapshot build compatibility (`goreleaser build --snapshot --clean`).
