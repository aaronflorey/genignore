# 04-01 Summary

- Expanded `internal/app/json_test.go` to assert stable `detect --json` and `add --json` payloads through the CLI, including command metadata, ordered selections, warnings, detection results, file action, and template counts.
- Expanded `internal/app/service_test.go` with dry-run regression coverage proving both `detect` and `add` execute the normal success path while leaving existing `.gitignore` content unchanged.
- Kept `internal/app/types.go` as the shared automation payload definition and preserved deterministic JSON serialization by continuing to marshal typed structs directly.
- Verification: `go test ./internal/app -run "Test(JSON|Detect|Add|List|Search)"`, `go test ./internal/app`.
