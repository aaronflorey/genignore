<!-- generated-by: gsd-doc-writer -->
# Development

## Local setup

1. Fork the repository in GitHub, then clone your fork:

```bash
git clone <your-fork-url>
cd genignore
```

2. Ensure your Go toolchain satisfies the module requirement (`go 1.22` in `go.mod`).

3. Download and tidy dependencies:

```bash
go mod tidy
```

4. Build and run tests locally before opening a PR:

```bash
go build ./...
go test ./...
```

5. Run the CLI during development from source:

```bash
go run . detect --dry-run
```

## Build commands

This repository does not use `package.json` scripts; development uses standard Go commands plus CI/release tooling.

| Command | Description |
| --- | --- |
| `go build ./...` | Compile all packages in the module. |
| `go test ./...` | Run the full test suite across all packages. |
| `go run . detect` | Run the CLI directly from source for local validation. |
| `go run . add <keys...>` | Test add-flow behavior against a working directory. |
| `goreleaser check` | Validate `.goreleaser.yaml` configuration (mirrors CI release-validation). |
| `goreleaser build --snapshot --clean` | Build local release artifacts without publishing. |

## Code style

- **Formatting/lint baseline:** Go standard formatting and idiomatic style (`gofmt`/`go fmt`) are expected for all changed files.
- **CI-enforced linting:** GitHub Actions runs `golangci/golangci-lint-action@v8` in `.github/workflows/ci.yml` (job: `lint-and-test`, step: `Lint`).
- **Project lint config:** No repository-local `.golangci.yml`/`.golangci.yaml` file is present; the workflow action configuration is the active lint entry point.
- **Tests in style gate:** CI also runs `go test ./...` in the same `lint-and-test` job.

## Branch conventions

- Default branch is `main` (release workflow triggers on pushes to `main` in `.github/workflows/release-please.yml`).
- No branch naming convention is documented in `CONTRIBUTING.md` or a PR template (neither file exists in this repository).

## PR process

- Open a pull request against `main`.
- Ensure CI is passing for your branch (`lint-and-test` and `release-validation` in `.github/workflows/ci.yml`).
- Use conventional commit prefixes such as `feat`, `fix`, `perf`, `docs`, `test`, `refactor`, or `chore` so `release-please` can categorize release notes (`release-please-config.json`).
- Keep changes scoped and include/adjust tests when behavior changes.
- If your change affects release packaging, verify `.goreleaser.yaml` still passes `goreleaser check`.
