# 03-01 Summary

- Hardened `internal/app/service.go` so `detect` iterates detectors in sorted key order, returns sorted `DetectionResults`, and fails before template fetch or file writes when include/exclude resolution produces an empty final provider set.
- Expanded `internal/app/service_test.go` with regression coverage for `detected + include - exclude`, exclude-over-include precedence, sorted selection slices, empty-result hard failure, and explicit detection-result ordering.
- Added `internal/provider/detectors_test.go` to cover file, OS, PATH, installed-app, and structured read/parse failure detector behavior.
- Refined `internal/provider/detectors.go` so read failures surface structured `Reason`, `Evidence`, and `Error` metadata instead of collapsing to generic non-matches.
- Verification: `go test ./internal/app -run TestDetect`, `go test ./internal/provider`.
