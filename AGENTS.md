# AGENTS.md

## Project Overview

`genignore` is a Rust CLI. It detects providers for the current directory, loads embedded `.gitignore` template content from the checked-in `github/gitignore` snapshot plus local custom templates, and updates only the managed marker block in `.gitignore`.

## Source Of Truth

- Workspace and crate versions: `Cargo.toml`, `Cargo.lock`
- CLI entrypoint: `crates/genignore-cli/src/main.rs` -> `crates/genignore-cli/src/spec.rs` (usage-lib grammar) -> `genignore_core::Service`
- Managed block behavior: `crates/genignore-core/src/manager.rs`
- Provider detection: `crates/genignore-detection/`, `internal/rulecatalog/rules.json`
- Git working-tree root gate: `crates/genignore-detection/src/gitroot.rs`
- Embedded upstream templates: git submodule `internal/templatecatalog/github-gitignore` from `.gitmodules` (compiled in by `crates/genignore-core/build.rs`)
- Local custom templates: `internal/customtemplate/templates/`
- CI and release automation: `.github/workflows/ci.yml`, `.github/workflows/release.yaml`, `release-please-config.json`
- Detailed docs: `docs/DEVELOPMENT.md`, `docs/TESTING.md`, `docs/ARCHITECTURE.md`, `docs/CONFIGURATION.md`

## Commands

| Purpose | Command | When to run |
| --- | --- | --- |
| Fetch deps | `cargo fetch` | Fresh checkout or dependency changes |
| Format | `cargo fmt --all` | After Rust edits |
| Build | `cargo build --workspace` | After CLI or crate changes |
| Test all | `cargo test --workspace` | Before handoff when practical; this also validates embedded catalog assets |
| Test one crate | `cargo test -p genignore-core` | Focused crate iteration |
| Test one case | `cargo test -p genignore-core --test contracts detect_diff_next_vscode_app_contract` | Focused test iteration |
| Lint | `cargo clippy --workspace --all-targets -- -D warnings` | Before handoff; matches CI |
| Format check | `cargo fmt --all -- --check` | Before handoff; matches CI |
| Run CLI from source | `cargo run -p genignore-cli -- detect --dry-run` | Manual smoke test without writing `.gitignore` |
| Release build | `cargo build --release -p genignore-cli` | Release config or packaging changes |

## Development Notes

- There is no npm-style task runner; use Cargo commands directly. `hk` runs the configured linters (`cargo_fmt`, `cargo_clippy`, secrets/whitespace checks).
- CI checks out submodules recursively. If template snapshot tests fail locally, confirm `internal/templatecatalog/github-gitignore` is initialized and in sync (`git submodule update --init --recursive`).
- Normal CLI execution must not require network access; provider catalogs, template bodies, and repository detection rules are embedded.
- Machine config is optional and strict TOML at `$HOME/.config/genignore/config.toml`; only `[defaults]`, `providers`, and `ignore_rules` are supported.
- `detect`, `add`, `resolve`, and `doctor` only run when the current directory is the top-level of a Git working tree (`git rev-parse --show-toplevel` canonicalized equals canonicalized cwd). `list` and `search` work anywhere.

## Testing And Fixtures

- Tests use Rust's built-in harness: `#[cfg(test)]` modules next to implementation plus `crates/<crate>/tests/` behavioral tests.
- Reuse fixture helpers in `crates/genignore-core/tests/contracts.rs` for contract tests; they copy fixtures to temp dirs and `git init` before asserting Git-root behavior.
- Keep `testdata/repos/` fixtures minimal, detector-relevant, and secret-free.
- `testdata/contracts/` contains machine-readable output contracts; update only for intentional output changes, then review provider ordering, provenance, and formatting churn.
- For intentional fixture-backed contract changes, run `cargo test --workspace` before accepting the diff.

## Invariants To Preserve

- Only content between `# BEGIN genignore` and `# END genignore` is CLI-owned; preserve user content outside markers byte-for-byte (including CRLF and missing final newline).
- Provider ordering and generated output must stay deterministic and alphabetically stable.
- Scope is the current directory only (repository root plus immediate, non-gitignored subdirectories); do not add monorepo traversal or plugin loading without an explicit product change.
- Managed blocks must include deterministic provenance for upstream and embedded custom providers.
- Required env rules are normalized by `crates/genignore-core/src/manager.rs`: `.env`, `.env.*`, `!.env.example`, `!.env.ci`.
- `genignore-detection` must not depend on `genignore-core` or `genignore-cli`.

## Catalog And Template Changes

- To add a custom provider, add `internal/customtemplate/templates/<key>.gitignore` and register it in `crates/genignore-core/src/catalog.rs` (`CUSTOM_TEMPLATES`), then add/update tests.
- To change repository-backed detection, edit `internal/rulecatalog/rules.json` and keep the strict-schema tests in `crates/genignore-detection` passing.
- Treat upstream `github/gitignore` snapshot refreshes as narrow maintenance changes because they can churn provider catalogs and generated contract output.

## Release Notes

- Release PRs are managed by release-please on `main` (`release-type: rust`); conventional commit types in `release-please-config.json` affect changelog sections.
- Release artifacts are matrix cargo builds in `.github/workflows/release.yaml` (`genignore_<version>_<os>_<arch>.{tar.gz,zip}` + `.sha256`), uploaded to the GitHub release; Homebrew publishing expects `HOMEBREW_TAP_GITHUB_TOKEN` only in release CI; never add real tokens to the repo.
- For release artifact/offline behavior checks, follow `docs/TESTING.md#manual-offline-release-verification`.
