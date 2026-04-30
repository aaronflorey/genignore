<!-- generated-by: gsd-doc-writer -->
# Testing

## Test framework and setup

This project uses Go's standard `testing` package for unit and integration-style tests, with `net/http/httptest` used for HTTP client tests (for example in `internal/api/client_test.go`). The module targets `go 1.22` (`go.mod`), so install a compatible Go toolchain and run:

```bash
go mod tidy
```

before running tests.

## Running tests

Run the full test suite:

```bash
go test ./...
```

Run tests for a single package:

```bash
go test ./internal/app
```

Run a single test by name:

```bash
go test ./internal/app -run TestListCommand
```

Run tests with coverage output:

```bash
go test ./... -coverprofile=coverage.out
```

Watch mode is not configured in this repository.

## Writing new tests

- Use Go's colocated naming convention: place tests next to implementation files and name them `*_test.go` (for example `internal/provider/detectors_test.go`, `internal/gitignore/manager_test.go`).
- Prefer table-driven subtests with `t.Run(...)` for multiple scenarios in one function (used extensively in `internal/provider/detectors_test.go`).
- Use `t.Parallel()` where tests are independent.
- Reuse existing test helpers when available, such as `captureRunOutput(...)` / `captureRunOutputWithHome(...)` in `internal/app/cli_test.go` for CLI-output assertions.
- Keep network behavior deterministic by using `httptest.NewServer` plus fixture files under `internal/api/testdata/`.

## Coverage requirements

No minimum coverage threshold is configured in this repository.

| Type | Threshold |
| --- | --- |
| Lines | Not configured |
| Branches | Not configured |
| Functions | Not configured |
| Statements | Not configured |

## CI integration

Tests run in GitHub Actions workflow `.github/workflows/ci.yml`:

- **Workflow:** `ci`
- **Triggers:** `push`, `pull_request`
- **Job:** `lint-and-test`
- **Test step command:** `go test ./...`

The same workflow also runs linting in the `lint-and-test` job before the test step.
