# 04-02 Summary

- Refined `internal/app/cli.go` so human-readable `detect` and `add` output stays compact and labeled, with provider selections rendered as concise comma-separated summaries and verbose detection evidence still gated behind `--verbose`.
- Updated catalog rendering in `internal/app/cli.go` so `list` and `search` expose a stable `Command`/`Query`/`Providers` structure in human output while `--json` continues to serialize `CatalogResult` directly.
- Expanded `internal/app/cli_test.go` and `internal/app/json_test.go` with contract coverage for detect/add/list/search stdout and JSON behavior, including warning visibility, dry-run file action visibility, and deterministic provider ordering.
- Verification: `go test ./internal/app -run "Test(JSON|Detect|Add|List|Search)"`, `go test ./internal/app`.
