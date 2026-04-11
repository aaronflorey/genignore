---
quick: 260411-wji
type: execute
autonomous: true
files_modified:
  - internal/provider/detectors.go
  - internal/provider/detectors_test.go
  - internal/provider/supported.go
requirements:
  - quick-provider-support-expansion
---

<objective>
Add support for requested providers in auto-detection and supported key validation, with tests proving stable behavior.
</objective>

<context>
@.planning/STATE.md
@internal/provider/detectors.go
@internal/provider/detectors_test.go
@internal/provider/supported.go
@.planning/quick/260411-wji-add-support-for-terraform-rust-java-kotl/260411-wji-CONTEXT.md
@.planning/quick/260411-wji-add-support-for-terraform-rust-java-kotl/260411-wji-RESEARCH.md
</context>

<tasks>
<task type="auto" tdd="true">
  <name>Add detector tests for requested providers</name>
  <files>internal/provider/detectors_test.go</files>
  <verify>
    <automated>go test ./internal/provider</automated>
  </verify>
</task>

<task type="auto" tdd="true">
  <name>Implement requested detectors and helper logic</name>
  <files>internal/provider/detectors.go</files>
  <verify>
    <automated>go test ./internal/provider</automated>
  </verify>
</task>

<task type="auto">
  <name>Add missing supported keys for parity</name>
  <files>internal/provider/supported.go</files>
  <verify>
    <automated>go test ./...</automated>
  </verify>
</task>
</tasks>

<verification>
- go test ./internal/provider
- go test ./...
</verification>

<success_criteria>
- All requested providers are auto-detectable via deterministic project signals.
- Supported key list includes requested keys that were previously absent.
- Provider and full test suites pass.
</success_criteria>

<output>
After completion, create `.planning/quick/260411-wji-add-support-for-terraform-rust-java-kotl/260411-wji-SUMMARY.md`.
</output>
