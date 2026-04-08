# 03-02 Summary

- Updated `internal/app/cli.go` so detect output stays concise by default, while `--verbose` renders deterministic matched/error detection evidence lines and `--json` continues to emit the structured `CommandResult` payload.
- Added a small CLI service injection seam in `internal/app/cli.go` so detect command tests stay offline and deterministic.
- Expanded `internal/app/cli_test.go` with detect command coverage for default output, verbose evidence rendering, and non-zero failure behavior.
- Expanded `internal/app/json_test.go` to assert the structured `detectionResults` JSON shape directly, including `key`, `matched`, `reason`, `evidence`, and `error` fields.
- Verification: `go test ./internal/app -run TestJSON`, `go test ./internal/app`, `go test ./...`.
