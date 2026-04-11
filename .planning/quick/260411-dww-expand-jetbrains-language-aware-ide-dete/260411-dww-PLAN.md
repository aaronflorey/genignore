---
quick: 260411-dww
type: execute
autonomous: true
files_modified:
  - internal/provider/detectors.go
  - internal/provider/detectors_test.go
requirements:
  - quick-jetbrains-language-aware-ide-expansion
must_haves:
  truths:
    - "When a JetBrains installation is detected, language signals infer JetBrains IDE providers for Python, JavaScript/TypeScript, Ruby, .NET, and C/C++ projects (pycharm, webstorm, rubymine, rider, clion)."
    - "Auto-detection includes every JetBrains IDE provider key we claim to support in this codebase (no supported JetBrains IDE key is silently undetectable)."
    - "Inference remains deterministic and only emits supported provider keys with explicit reasons/evidence."
  artifacts:
    - path: "internal/provider/detectors.go"
      provides: "Expanded JetBrains inference mapping + detector registry coverage for supported JetBrains IDE keys"
    - path: "internal/provider/detectors_test.go"
      provides: "Regression tests for language-aware inference and supported-key coverage"
  key_links:
    - from: "JetBrains install detector result"
      to: "IDE-specific provider match"
      via: "language-signal inference fallback"
      pattern: "jetbrains matched + language signal => ide key matched"
    - from: "SupportedKeys (JetBrains subset)"
      to: "Registry()"
      via: "detector registration"
      pattern: "supported JetBrains keys are present in detector registry"
---

<objective>
Expand JetBrains language-aware IDE detection so installed JetBrains tooling can infer pycharm, webstorm, rubymine, rider, and clion from project language signals, and ensure all supported JetBrains IDE providers are covered by auto-detection.

Purpose: reduce false negatives in provider detection for common JetBrains-based workflows across languages.
Output: updated provider detector mapping and focused regression tests.
</objective>

<context>
@.planning/STATE.md
@AGENTS.md
@internal/provider/detectors.go
@internal/provider/detectors_test.go
@internal/provider/supported.go
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add failing tests for JetBrains language-aware inference expansion and supported-key coverage</name>
  <files>internal/provider/detectors_test.go</files>
  <behavior>
    - Test 1: With JetBrains present and Python signals, `pycharm` is inferred even when direct PyCharm install path is absent.
    - Test 2: With JetBrains present and JS/TS signal (`package.json`), `webstorm` is inferred when direct install path is absent.
    - Test 3: With JetBrains present and language signals for Ruby (`Gemfile`), .NET (`*.sln` or `*.csproj`), and C/C++ (`CMakeLists.txt`), infer `rubymine`, `rider`, and `clion` respectively.
    - Test 4: Add a coverage assertion that all supported JetBrains IDE keys in `supported.go` that should be auto-detectable are present in `Registry()` (catching missing providers such as `androidstudio` if absent).
    - Test 5: Without matching language signal, new inference paths do not match.
  </behavior>
  <action>Create deterministic unit tests using temp dirs and injected install candidates, following existing JetBrains inference test style. Keep assertions on full `Result` payload (key, matched, reason, evidence) and avoid host-environment dependence.</action>
  <verify>
    <automated>go test ./internal/provider -run "Test(JetBrains|IDE|Registry|AppDetector)"</automated>
  </verify>
  <done>Tests encode the requested inference behavior and supported-provider coverage and fail before implementation.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement expanded JetBrains inference map and add missing supported JetBrains IDE detectors</name>
  <files>internal/provider/detectors.go</files>
  <behavior>
    - `pycharm`, `webstorm`, `rubymine`, `rider`, and `clion` support inference from JetBrains install + language signal.
    - Missing supported JetBrains IDE provider keys are added to install candidates/registry so they can be auto-detected.
    - Existing behavior for direct IDE path detection and phpstorm/goland inference remains intact.
  </behavior>
  <action>Refactor the inference helper to support per-IDE signal rules (single-file and multi-pattern as needed), wire those detectors in `Registry()`, and add/adjust install candidate lists for any supported JetBrains IDE provider key currently missing detector coverage. Preserve deterministic ordering and structured reasons/evidence.</action>
  <verify>
    <automated>go test ./internal/provider -run "Test(JetBrains|IDE|Registry|AppDetector)"</automated>
  </verify>
  <done>Expanded inference and missing-provider detector coverage are implemented and all added tests pass.</done>
</task>

<task type="auto">
  <name>Task 3: Run focused regression checks for detect pipeline stability</name>
  <files>internal/provider/detectors.go, internal/provider/detectors_test.go</files>
  <action>Run provider and app-layer targeted tests to confirm new JetBrains inference rules integrate cleanly with detect output and do not regress existing detectors.</action>
  <verify>
    <automated>go test ./internal/provider ./internal/app -run "Test(Detect|JetBrains|IDE|Registry|AppDetector|PathDetector|Vue|React|Laravel)"</automated>
  </verify>
  <done>Targeted suites pass with stable, deterministic detection behavior.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Local filesystem -> provider detector results | Untrusted local project files/install paths influence inferred provider keys and resulting API template requests |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-260411-dww-01 | T | `internal/provider/detectors.go` JetBrains inference rules | mitigate | Gate inference on explicit language signals and JetBrains install evidence; add negative tests for no-signal scenarios |
| T-260411-dww-02 | D | Registry/provider selection stability | mitigate | Keep deterministic candidate evaluation and add registry-coverage tests for supported JetBrains IDE keys |
| T-260411-dww-03 | I | Detector evidence reporting | accept | Evidence exposes local path metadata already used by existing detector diagnostics and does not read file contents |
</threat_model>

<verification>
- `go test ./internal/provider -run "Test(JetBrains|IDE|Registry|AppDetector)"`
- `go test ./internal/provider ./internal/app -run "Test(Detect|JetBrains|IDE|Registry|AppDetector|PathDetector|Vue|React|Laravel)"`
</verification>

<success_criteria>
- JetBrains presence plus language signals infers `pycharm`, `webstorm`, `rubymine`, `rider`, and `clion` when direct app install paths are unavailable.
- Any supported JetBrains IDE provider key intended for auto-detection is represented in `Registry()` with install candidates.
- Focused provider/app regression tests pass with deterministic output semantics.
</success_criteria>

<output>
After completion, create `.planning/quick/260411-dww-expand-jetbrains-language-aware-ide-dete/260411-dww-SUMMARY.md`.
</output>
