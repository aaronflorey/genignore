# 05-02 Summary

- Replaced the generic Vite-based Vue detection path in `internal/provider/detectors.go` with Vue-specific config and `package.json` dependency checks, then expanded `internal/provider/detectors_test.go` to cover the Vue-on-Vite false-positive regression and structured detector outcomes.
- Expanded `internal/app/service_test.go`, `internal/app/cli_test.go`, and `internal/app/json_test.go` to assert deterministic selection and warning ordering, stable failure behavior, and consistent human/JSON output contracts under richer warning scenarios.
- Kept `internal/app/types.go` and the existing command result contract intact while broadening regression coverage around the shipped phase 5 behaviors.
- Verification: `go test ./internal/provider`, `go test ./internal/app`, `go test ./...`.
