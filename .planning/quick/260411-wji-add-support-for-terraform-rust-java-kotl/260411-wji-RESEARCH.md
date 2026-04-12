# Quick Task 260411-wji Research

## Goal

Identify low-noise project signals for requested provider keys that fit `genignore` detector conventions.

## Findings

- Existing detectors prefer simple filesystem signals and deterministic ordering.
- Language/framework providers requested here map cleanly to conventional files already used by tooling ecosystems.
- `flutter` requires stricter handling than `dart` to avoid false positives because both use `pubspec.yaml`.

## Recommended Signal Map

- `terraform`: `*.tf`, `*.tfvars`, `.terraform.lock.hcl`
- `rust`: `Cargo.toml`
- `java`: `pom.xml`, `build.gradle`, `build.gradle.kts`
- `kotlin`: `build.gradle.kts`, `settings.gradle.kts`, `*.kt`
- `dotnetcore`: `*.sln`, `*.csproj`
- `csharp`: `*.sln`, `*.csproj`, `*.cs`
- `dart`: `pubspec.yaml`
- `flutter`: `pubspec.yaml` containing `flutter:` or `sdk: flutter`
- `swift`: `Package.swift`, `*.swift`
- `xcode`: `*.xcodeproj`, `*.xcworkspace`
- `android`: `AndroidManifest.xml` (root or `app/src/main`)
- `ruby`: `Gemfile`
- `maven`: `pom.xml`
- `rails`: `bin/rails`, `config/application.rb`
- `jekyll`: `_config.yml`
- `symfony`: `bin/console`, `config/bundles.php`, `symfony.lock`

## Risks and Mitigations

- False positives from broad globs: mitigate by favoring canonical files and pairing with framework markers where needed.
- Overlap between providers (Java/Kotlin, Dart/Flutter): mitigate with provider-specific matching and dedicated Flutter parsing.

## Outcome

Proceed with deterministic detector additions, focused tests, and supported-key updates for parity.
