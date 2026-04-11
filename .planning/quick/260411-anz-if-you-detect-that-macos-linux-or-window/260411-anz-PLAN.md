---
quick: 260411-anz
type: execute
autonomous: true
files_modified:
  - internal/app/service.go
  - internal/app/service_test.go
requirements:
  - quick-detect-preserve-existing-os-providers
must_haves:
  truths:
    - "When detect runs, OS providers already present in the managed block (`macos`, `linux`, `windows`) are preserved instead of being dropped just because current runtime differs."
    - "Non-OS providers from previous managed block are still reset by detect unless currently detected or explicitly included."
    - "Provider ordering remains alphabetical and deterministic in API request + managed block metadata."
  artifacts:
    - path: "internal/app/service.go"
      provides: "Detect flow that unions previously managed OS providers with current detect/include set"
    - path: "internal/app/service_test.go"
      provides: "Regression tests for cross-OS detect preservation and reset boundaries"
  key_links:
    - from: "Manager.ReadManagedProviders()"
      to: "Detect final provider selection"
      via: "OS-only carry-forward filter before template fetch"
      pattern: "existing managed providers intersect {macos,linux,windows}"
    - from: "finalProviders"
      to: "Client.FetchTemplate + BuildManagedBlock"
      via: "sorted provider slice"
      pattern: "mapKeysSorted output reused end-to-end"
---

<objective>
Adjust `detect` so previously managed OS templates are retained across cross-platform runs (macOS/Linux/Windows) while keeping existing detect-reset behavior for non-OS providers.

Purpose: prevent teammates on different operating systems from unintentionally removing each other’s OS ignore rules during normal detect usage.
Output: detect selection logic update plus targeted regression tests.
</objective>

<context>
@.planning/STATE.md
@AGENTS.md
@internal/app/service.go
@internal/app/service_test.go
@internal/gitignore/manager.go
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add detect regression tests for preserving previously managed OS providers</name>
  <files>internal/app/service_test.go</files>
  <behavior>
    - Test 1: Existing managed block with `macos` stays in final providers when current detect run matches `linux` (cross-OS union).
    - Test 2: Existing non-OS provider (e.g., `python`) is still dropped when not currently detected/included.
    - Test 3: Explicit `--exclude macos` (DetectOptions.Exclude) still removes preserved OS provider.
  </behavior>
  <action>Replace/adjust the current detect-reset expectation to encode the new OS-preservation rule. Seed `.gitignore` managed block via `BuildManagedBlock`, run `Detect`, and assert both `res.FinalProviders` and `fakeAPI.requests[0]` are sorted and match expected preserved OS behavior.</action>
  <verify>
    <automated>go test ./internal/app -run "TestDetect.*(Preserve|Reset|Exclude)"</automated>
  </verify>
  <done>Tests fail against current behavior and precisely define OS-only carry-forward semantics.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement OS-provider carry-forward in Detect selection pipeline</name>
  <files>internal/app/service.go</files>
  <behavior>
    - Previously managed `macos`/`linux`/`windows` are added to detect final set before exclude processing.
    - Non-OS managed providers are not auto-carried.
    - Existing include/exclude precedence and deterministic sorting remain unchanged.
  </behavior>
  <action>In `Service.Detect`, read existing managed providers (`s.Manager.ReadManagedProviders()`), filter to OS keys only, and union them into the `final` set alongside detected + include keys; then apply excludes so user intent can still remove any key. Keep implementation local to `service.go` with explicit constant/set for allowed OS carry-forward keys.</action>
  <verify>
    <automated>go test ./internal/app -run "TestDetect.*(Preserve|Reset|Exclude)"</automated>
  </verify>
  <done>Detect now preserves prior managed OS providers across machine differences without regressing non-OS reset behavior.</done>
</task>

<task type="auto">
  <name>Task 3: Run focused app + gitignore verification for stability</name>
  <files>internal/app/service.go, internal/app/service_test.go</files>
  <action>Run the relevant package tests to ensure detect/add flows and managed-block interactions stay stable after OS carry-forward changes.</action>
  <verify>
    <automated>go test ./internal/app ./internal/gitignore</automated>
  </verify>
  <done>Targeted suites pass, confirming cross-OS preservation and no managed-block regression.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Existing `.gitignore` managed metadata -> provider selection | Previously written provider list influences new remote template fetch and generated ignore rules |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-260411-anz-01 | T | `Service.Detect` provider merge logic | mitigate | Restrict carry-forward to explicit allowlist `{macos,linux,windows}` and test that non-OS keys are not preserved |
| T-260411-anz-02 | D | `.gitignore` churn across OS contributors | mitigate | Add regression test proving cross-OS detect keeps prior OS keys and remains sorted/deterministic |
| T-260411-anz-03 | R | User override semantics | mitigate | Apply `Exclude` after carry-forward and assert in tests that excluded OS key is removed |
</threat_model>

<verification>
- `go test ./internal/app -run "TestDetect.*(Preserve|Reset|Exclude)"`
- `go test ./internal/app ./internal/gitignore`
</verification>

<success_criteria>
- Detect no longer removes previously managed `macos`/`linux`/`windows` solely due to current runtime OS.
- Detect continues resetting non-OS managed providers unless currently selected by detect/include.
- Final provider list and API request remain alphabetically stable.
</success_criteria>

<output>
After completion, create `.planning/quick/260411-anz-if-you-detect-that-macos-linux-or-window/260411-anz-SUMMARY.md`.
</output>
