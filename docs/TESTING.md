<!-- generated-by: gsd-doc-writer -->
# Testing

## Test framework and setup

This repository uses Go's standard `testing` package. The module requires Go `1.22` (`go.mod`), so install a compatible Go toolchain and download dependencies before running tests:

```bash
go mod download
```

## Running tests

Run the full suite:

```bash
go test ./...
```

Run one package:

```bash
go test ./internal/provider
```

Run one test by name:

```bash
go test ./internal/app -run TestListCommand
```

Run with coverage output:

```bash
go test ./... -coverprofile=coverage.out
```

Watch mode is not configured in this repository.

## Writing new tests

- Keep tests colocated with implementation and use the `*_test.go` pattern (for example `internal/app/service_test.go`, `internal/gitignore/manager_test.go`).
- Prefer table-driven tests with `t.Run(...)` for multi-case behavior (for example in `internal/provider/detectors_test.go`).
- Use `t.Parallel()` for independent tests to reduce suite runtime.
- Reuse CLI helpers in `internal/app/cli_test.go`, including `captureRunOutput(...)` and `captureRunOutputWithHome(...)`, for command-output assertions.
- Prefer embedded-fixture tests for provider catalogs, template content, and JSON rule-catalog behavior so runtime network assumptions do not leak back into the suite.

## Coverage requirements

No explicit coverage threshold is configured in this repository.

| Type | Threshold |
| --- | --- |
| Lines | Not configured |
| Branches | Not configured |
| Functions | Not configured |
| Statements | Not configured |

## CI integration

Tests run in GitHub Actions via `.github/workflows/ci.yml`.

- **Workflow:** `ci`
- **Triggers:** `push`, `pull_request`
- **Job:** `lint-and-test`
- **Test command:** `go test ./...`

The same job runs linting (`golangci/golangci-lint-action@v8`) before the test step.
