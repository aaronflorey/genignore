---
quick: 260411-dnn
type: execute
autonomous: true
files_modified:
  - internal/provider/detectors.go
  - internal/provider/detectors_test.go
requirements:
  - quick-cross-platform-ide-detection
must_haves:
  truths:
    - "Detect identifies installed IDE providers on both macOS and Linux (not only `/Applications/*`)."
    - "When a JetBrains install is present, project language signals infer IDE provider keys (`phpstorm` for PHP/composer projects, `goland` for Go projects)."
    - "IDE detection remains deterministic and does not add unsupported provider keys."
  artifacts:
    - path: "internal/provider/detectors.go"
      provides: "Cross-platform IDE install probing + language-aware JetBrains inference detector logic"
    - path: "internal/provider/detectors_test.go"
      provides: "Regression tests for macOS/Linux install paths and JetBrains language inference"
  key_links:
    - from: "Registry()"
      to: "Detect pipeline in internal/app/service.go"
      via: "provider detector map entries for phpstorm/goland/jetbrains"
      pattern: "Registry includes inferred JetBrains IDE detectors"
    - from: "project signal files (composer.json/go.mod)"
      to: "IDE provider match result"
      via: "language-signal gate inside JetBrains detector"
      pattern: "language signal + installed JetBrains => Matched true"
---

<objective>
Add cross-platform installed-IDE detection for macOS and Linux, and make JetBrains IDE detection language-aware so project signals infer the most relevant IDE templates.

Purpose: improve provider auto-detection quality for real developer environments where JetBrains IDEs are installed outside fixed macOS app paths.
Output: detector logic updates plus focused regression tests.
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
  <name>Task 1: Add failing detector tests for Linux/macOS IDE installs and JetBrains language inference</name>
  <files>internal/provider/detectors_test.go</files>
  <behavior>
    - Test 1: IDE install probing matches PhpStorm for both macOS-style and Linux-style candidate locations.
    - Test 2: JetBrains detection infers `phpstorm` when project has PHP signal (`composer.json`) and a JetBrains install is present.
    - Test 3: JetBrains detection infers `goland` when project has `go.mod` and a JetBrains install is present.
    - Test 4: Without matching language signal, JetBrains inference does not claim language-specific IDE keys.
  </behavior>
  <action>Create table-driven unit tests around detector helpers/registry behavior using temp dirs and injected candidate paths (no dependence on host machine IDE installs). Keep assertions explicit on `Result{Key, Matched, Reason, Evidence}` and preserve deterministic expectations.</action>
  <verify>
    <automated>go test ./internal/provider -run "Test(IDE|JetBrains|AppDetector)"</automated>
  </verify>
  <done>Tests clearly encode desired cross-platform install detection and language-aware inference behavior and fail before implementation.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement cross-platform install probing and JetBrains language-aware IDE inference</name>
  <files>internal/provider/detectors.go</files>
  <behavior>
    - `phpstorm`/`goland` detection can match both macOS and Linux install path conventions.
    - JetBrains language inference maps project signals to IDE keys (`composer.json` -> `phpstorm`, `go.mod` -> `goland`) when JetBrains is installed.
    - Existing detector semantics stay structured (clear reason/evidence, no unsupported key emission).
  </behavior>
  <action>Refactor IDE detection helpers to support OS-aware candidate paths and add a JetBrains inference detector that gates on both (a) JetBrains install presence and (b) project language signals. Keep provider keys aligned with `SupportedKeys`, preserve existing detector contracts, and avoid introducing nondeterministic filesystem scans.</action>
  <verify>
    <automated>go test ./internal/provider -run "Test(IDE|JetBrains|AppDetector)"</automated>
  </verify>
  <done>Detector implementation satisfies all new tests and produces consistent provider keys for macOS/Linux plus JetBrains language inference.</done>
</task>

<task type="auto">
  <name>Task 3: Run focused provider + app detection regression suite</name>
  <files>internal/provider/detectors.go, internal/provider/detectors_test.go</files>
  <action>Run relevant package tests to confirm detector changes do not regress existing project/language/runtime detection behavior.</action>
  <verify>
    <automated>go test ./internal/provider ./internal/app -run "Test(Detect|AppDetector|PathDetector|Vue|React|Laravel|JetBrains|IDE)"</automated>
  </verify>
  <done>Targeted suites pass, confirming IDE enhancements are stable and integrate with detect flow.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Local filesystem -> provider detector results | Untrusted filesystem state (files/paths) influences provider selection and outgoing template request |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-260411-dnn-01 | T | `internal/provider/detectors.go` IDE path matching | mitigate | Restrict inference to explicit IDE keys and explicit language signals (`composer.json`, `go.mod`) with test coverage for false positives |
| T-260411-dnn-02 | D | Detect provider output stability | mitigate | Keep deterministic candidate evaluation + assert stable `Result` shape and provider keys in tests |
| T-260411-dnn-03 | I | Detector evidence paths in output | accept | Evidence is local path metadata only and already part of existing detect diagnostics; no secret file contents are read |
</threat_model>

<verification>
- `go test ./internal/provider -run "Test(IDE|JetBrains|AppDetector)"`
- `go test ./internal/provider ./internal/app -run "Test(Detect|AppDetector|PathDetector|Vue|React|Laravel|JetBrains|IDE)"`
</verification>

<success_criteria>
- Running detect on macOS or Linux can identify installed IDE providers using platform-appropriate install path logic.
- JetBrains language-aware inference selects PhpStorm for PHP projects and GoLand for Go projects when JetBrains installation is detected.
- Provider detection remains deterministic and existing detect flows stay green in focused regression tests.
</success_criteria>

<output>
After completion, create `.planning/quick/260411-dnn-add-detection-for-installed-ides-on-both/260411-dnn-SUMMARY.md`.
</output>
