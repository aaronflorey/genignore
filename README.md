# genignore

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/aaronflorey/genignore/actions/workflows/ci.yml/badge.svg)](https://github.com/aaronflorey/genignore/actions/workflows/ci.yml)
[![Release](https://github.com/aaronflorey/genignore/actions/workflows/release.yaml/badge.svg)](https://github.com/aaronflorey/genignore/actions/workflows/release.yaml)
[![Latest Release](https://img.shields.io/github/v/release/aaronflorey/genignore?sort=semver)](https://github.com/aaronflorey/genignore/releases/latest)

`genignore` is a Rust CLI for developers who want deterministic `.gitignore` generation while preserving manual rules outside a managed marker block.

## Installation

```bash
cargo install --git https://github.com/aaronflorey/genignore genignore-cli
```

If you use Homebrew:

```bash
brew install aaronflorey/tap/genignore
```

## Quick start

1. Detect providers in the current directory and create or update the managed `.gitignore` block:

```bash
genignore detect
```

2. Preview what would change without writing files:

```bash
genignore detect --dry-run
```

Or preview the exact managed-block diff without writing files:

```bash
genignore detect --diff
```

3. Add specific providers to the existing managed set:

```bash
genignore add go node
```

4. Optionally add machine-level defaults for extra providers or ignore rules:

```toml
[defaults]
providers = ["go", "node"]
ignore_rules = [".direnv/", "coverage.out"]
```

If you are working from source:

```bash
git clone https://github.com/aaronflorey/genignore.git
cd genignore
git submodule update --init --recursive
cargo run -p genignore-cli -- detect
```

## Usage examples

List all supported provider keys:

```bash
genignore list
```

Search providers by term:

```bash
genignore search jetbrains
```

Run detection with machine-readable JSON output:

```bash
genignore detect --json
```

Resolve detected and explicitly included providers without mutating `.gitignore`:

```bash
genignore resolve
genignore resolve --include macos --exclude windows --json
```

Explain the current detector evidence, provider resolution, embedded catalogs, and provenance decisions:

```bash
genignore doctor
genignore doctor --json
```

Exclude certain providers from detection:

```bash
genignore detect --exclude windows,macos
```

Machine-level configuration supports only the `defaults` table. Unknown fields are rejected, so stale `[runtime]` settings now fail config loading instead of being ignored.

The canonical supported-provider contract is the embedded `github/gitignore` catalog snapshot shipped with `genignore`, plus the embedded `ai-agents` and `wrangler` exceptions. Template bodies are loaded from checked-in content, so normal command execution does not require network access.

Generated managed blocks now include a deterministic `# Provenance:` line that records the pinned `github/gitignore` commit and any embedded providers that contributed content.

`genignore doctor` is the supported diagnostics surface for detector evidence, provider resolution, embedded catalog counts, JSON rule-catalog status, retained embedded custom providers, and managed-block provenance. Detection entries classify repository-backed evidence separately from host-only heuristics such as runtime OS or installed-application checks.

`genignore resolve` is the supported read-only automation surface for provider detection and final provider resolution. It reuses the same ordering, key validation, and include or exclude normalization as `genignore detect`, but it does not assemble or mutate the managed `.gitignore` block.

`genignore` still supports machine-level defaults only. It does not support per-project preset files or repository-local configuration in the current product scope. Any future preset concept remains explicitly deferred to a later scoped phase.

`genignore detect --diff` and `genignore add --diff` preview the exact managed-block change without writing `.gitignore`. The preview reports the same `File:` action that the eventual write path would take: `created`, `updated`, or `no-op`.

Output labels in human-readable mode include `Command:`, `Target:`, `Detected:`, `Final:`, `Added:`, `Included:`, `Excluded:`, `File:`, `Preview:`, `Diff:`, `Warning:`, `Detection:`, `Embedded catalog providers:`, `Selected providers:`, `Rule catalog:`, `Retained custom providers:`, `Decision:`, and `Provenance:`.

Default editor detection is intentionally repo-backed: `visualstudiocode` is detected from `.vscode/` or `*.code-workspace`, and `jetbrains` is detected from `.idea/` or `*.iml`. Installed editors alone do not change default detection results.

## Development

For local setup, build, test, lint, and release-verification commands, see [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md).

Quick reference:

```bash
cargo build --workspace      # compile all crates
cargo test --workspace       # run all tests
cargo run -p genignore-cli -- detect --dry-run  # smoke test without writing .gitignore
```

For testing details, see [`docs/TESTING.md`](docs/TESTING.md). For architecture overview, see [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## License

This project is licensed under the [MIT License](LICENSE).
