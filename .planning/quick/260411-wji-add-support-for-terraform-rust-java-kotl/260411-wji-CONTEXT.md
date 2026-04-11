# Quick Task 260411-wji: add support for terraform, rust, java, kotlin, dotnetcore, csharp, dart, flutter, swift, xcode, android, ruby, maven, rails, jekyll, symfony - Context

**Gathered:** 2026-04-11
**Status:** Ready for planning

<domain>
## Task Boundary

Add and validate support for the requested provider keys in the auto-detection pipeline and supported-key catalog.

</domain>

<decisions>
## Implementation Decisions

### Detection signals
- Use deterministic root-level file and glob signals that match existing detector style.

### Flutter behavior
- Match `flutter` only when `pubspec.yaml` contains Flutter markers to avoid matching plain Dart projects.

### Supported keys parity
- Add missing supported keys (`maven`, `rails`, `jekyll`) so `list`, `search`, and validation align with detection.

### the agent's Discretion
- Reuse existing detector helpers where possible and add minimal helper functions only when they reduce duplication.

</decisions>

<specifics>
## Specific Ideas

- Terraform: `*.tf`, `*.tfvars`, `.terraform.lock.hcl`.
- Kotlin: `build.gradle.kts`, `settings.gradle.kts`, or `*.kt`.
- Android: root and app-path Android manifests.
- Ruby/Rails/Symfony/Jekyll/Maven: canonical framework project files.

</specifics>
