# Contributing to genignore

Thanks for your interest in contributing to `genignore`! This guide covers the basics of getting set up and submitting changes.

## Prerequisites

- Go 1.22+ (see `go.mod` for the exact version)
- [mise](https://mise.jdx.dev/) for release tooling (`goreleaser`)
- Git with submodules enabled

## Local setup

1. Fork the repository, then clone your fork:

   ```bash
   git clone <your-fork-url>
   cd genignore
   ```

2. Initialize the embedded template submodule:

   ```bash
   git submodule update --init --recursive
   ```

3. Download Go module dependencies:

   ```bash
   go mod download
   ```

4. Verify your environment:

   ```bash
   go build ./...
   go test ./...
   ```

## Development workflow

Detailed build, test, lint, and release-verification commands are in [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md).

Quick reference:

| Purpose | Command |
| --- | --- |
| Build | `go build ./...` |
| Test all | `go test ./...` |
| Test one package | `go test ./internal/provider` |
| Run from source | `go run . detect --dry-run` |
| Lint (CI uses) | `golangci-lint run` |
| Validate release config | `mise x -- goreleaser check` |
| Snapshot release build | `mise x -- goreleaser release --snapshot --clean --skip=publish` |

## Commit conventions

This project uses [conventional commits](https://www.conventionalcommits.org/) for automated changelog generation via release-please:

- `feat:` new features
- `fix:` bug fixes
- `perf:` performance improvements
- `docs:` documentation changes (hidden from changelog)
- `test:` test changes (hidden from changelog)
- `refactor:` code refactoring (hidden from changelog)
- `chore:` maintenance tasks (hidden from changelog)

Use squash-merge when merging PRs for cleaner changelogs.

## Pull request process

1. Open your PR against `main`.
2. Ensure CI checks pass (`lint-and-test` and `release-validation` jobs in `.github/workflows/ci.yml`).
3. Keep PR scope focused and include or update tests when behavior changes.
4. If changes affect packaging, validate with `mise x -- goreleaser check` and `mise x -- goreleaser build --snapshot --clean`.

## Testing notes

- Tests use Go's standard `testing` package alongside implementation as `*_test.go`.
- `testdata/repos/` contains minimal fixture repos for detector testing &mdash; keep them small and secret-free.
- `testdata/contracts/` contains machine-readable output contracts &mdash; update only for intentional output changes.
- See [`docs/TESTING.md`](docs/TESTING.md) for detailed testing guidance.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
