# 05-01 Summary

- Updated `internal/api/client.go` and `internal/api/testdata/list.json` so offline API tests exercise the real list contract shape, while still tolerating the earlier `gitignores` array response during decoding.
- Expanded `internal/api/client_test.go`, `internal/gitignore/manager_test.go`, and `internal/app/service_test.go` to lock offline API failure handling, byte-exact `.gitignore` migration safety, dry-run non-write behavior, malformed marker preservation, and no-op reruns.
- Tightened `internal/gitignore/manager.go` so equivalent reruns avoid rewriting `.gitignore` content while keeping existing command behavior stable.
- Verification: `go test ./internal/api`, `go test ./internal/gitignore`, `go test ./internal/app`, `go test ./...`.
