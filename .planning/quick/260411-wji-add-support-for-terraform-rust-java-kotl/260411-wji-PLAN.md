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
must_haves:
  truths:
    - "Requested provider keys are detectable via `provider.Registry()` using deterministic file/glob signals."
    - "Requested provider keys are accepted by support validation via `provider.SupportedKeys` and `IsSupported`."
    - "Tests lock detector behavior including Flutter-specific gating to avoid Dart-only false positives."
  artifacts:
    - path: "internal/provider/detectors.go"
      provides: "Detector map entries and helper logic for requested provider detection"
    - path: "internal/provider/supported.go"
      provides: "Requested support keys present in provider catalog"
    - path: "internal/provider/detectors_test.go"
      provides: "Regression tests for requested providers and Flutter semantics"
  key_links:
    - from: "internal/provider/detectors.go Registry()"
      to: "internal/app/service.go Detect()"
      via: "sorted detector execution"
      pattern: "registry entry matched => detected provider included in final selection"
    - from: "internal/provider/supported.go SupportedKeys"
      to: "internal/app/service.go sanitizeKeys()"
      via: "provider.IsSupported"
      pattern: "key listed in SupportedKeys => include/add accepted"
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
  <action>Add deterministic tests that validate all requested provider detectors match canonical project signals, plus Flutter positive/negative behavior and registry-presence assertions.</action>
  <verify>
    <automated>go test ./internal/provider</automated>
  </verify>
  <done>Tests fail before implementation for missing detector coverage and pass once code is added.</done>
</task>

<task type="auto" tdd="true">
  <name>Implement requested detectors and helper logic</name>
  <files>internal/provider/detectors.go</files>
  <action>Add detector registrations and helper composition for requested providers while preserving existing deterministic behavior and reason/evidence conventions.</action>
  <verify>
    <automated>go test ./internal/provider</automated>
  </verify>
  <done>All requested keys have detector implementations and provider tests pass.</done>
</task>

<task type="auto">
  <name>Add missing supported keys for parity</name>
  <files>internal/provider/supported.go</files>
  <action>Add any requested keys absent from SupportedKeys so list/search/include validation and detection coverage stay aligned.</action>
  <verify>
    <automated>go test ./...</automated>
  </verify>
  <done>Supported catalog includes requested keys and full test suite remains green.</done>
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
