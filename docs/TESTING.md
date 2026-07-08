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

This command already covers the embedded-asset checks:

- `internal/templatecatalog` verifies the embedded `github-gitignore` snapshot loads, stays sorted, and matches the checked-out submodule files.
- `internal/rulecatalog` verifies the embedded `rules.json` catalog loads and rejects invalid catalog structure.
- `internal/app` contract tests exercise the embedded-backed CLI JSON contracts added for detect/resolve/doctor stability.

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

## Manual offline release verification

Use this check when validating a release artifact or a local build against the embedded, no-network success proof.

1. Build the binary:

```bash
go build -o ./dist/genignore-manual .
binary_path="$PWD/dist/genignore-manual"
```

2. Create a clean temporary repo and clean home/cache roots with no warm-up state:

```bash
tmp_root="$(mktemp -d)"
repo_dir="$tmp_root/repo"
home_dir="$tmp_root/home"
mkdir -p "$repo_dir" "$home_dir/.cache"
printf '{"name":"offline-check"}\n' > "$repo_dir/package.json"
```

3. Disable network access for the shell you use to run the binary (for example, airplane mode, a network-disabled VM/container, or Linux tools such as `bwrap --unshare-net` or `unshare -n`), then run generation with the clean HOME and cache paths:

```bash
(cd "$repo_dir" && HOME="$home_dir" XDG_CACHE_HOME="$home_dir/.cache" "$binary_path" detect --exclude linux,macos,windows)
```

4. Confirm the manual proof:

```bash
test -f "$repo_dir/.gitignore"
grep -q '# Provenance: github/gitignore@' "$repo_dir/.gitignore"
grep -q 'node_modules/' "$repo_dir/.gitignore"
test -z "$(ls -A "$home_dir/.cache")"
```

Expected result: the command succeeds with network disabled, writes a managed block for the detected `node` provider, and leaves the clean cache directory empty.

## Writing new tests

- Keep tests colocated with implementation and use the `*_test.go` pattern (for example `internal/app/service_test.go`, `internal/gitignore/manager_test.go`).
- Prefer table-driven tests with `t.Run(...)` for multi-case behavior (for example in `internal/provider/detectors_test.go`).
- Use `t.Parallel()` for independent tests to reduce suite runtime.
- Reuse CLI helpers in `internal/app/cli_test.go`, including `captureRunOutput(...)` and `captureRunOutputWithHome(...)`, for command-output assertions.
- Prefer embedded-fixture tests for provider catalogs, template content, and JSON rule-catalog behavior so network assumptions do not leak back into the suite.

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

No extra CI-only asset check is required today: the package tests above already make `go test ./...` fail when embedded catalog files or the rule catalog are missing or invalid.

The same job runs linting (`golangci/golangci-lint-action@v8`) before the test step.
