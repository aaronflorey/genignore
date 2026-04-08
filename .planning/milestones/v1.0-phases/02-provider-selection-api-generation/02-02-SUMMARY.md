# 02-02 Summary

- Added `internal/app/catalog.go` with deterministic provider discovery helpers: `ListProviders()` and case-insensitive `SearchProviders(term)`.
- Added `internal/app/catalog_test.go` to validate full-list ordering, substring search behavior, and empty-result behavior.
- Added `list` and `search <term>` commands in `internal/app/cli.go`, including JSON output via a dedicated `CatalogResult` payload.
- Added `internal/app/cli_test.go` to verify list/search command behavior and sorted JSON output.
- Verification: `go test ./internal/app ./internal/api`.
