# Contributing to genignore

Thanks for your interest in contributing to `genignore`! This guide covers the basics of getting set up and submitting changes.

## Prerequisites

- A current stable Rust toolchain (see `mise.toml` for the pinned version)
- [mise](https://mise.jdx.dev/) for the pinned toolchain and hook tooling (`rust`, `hk`)
- Git with submodules enabled

## Local setup

1. Fork the repository, then clone your fork:

   ```bash
   git clone <your-fork-url>
   cd genignore
   ```

2. Initialize the embedded template submodule:

   ```bash
   git submodule update --init --recursive
   ```

3. Fetch crate dependencies and verify your environment:

   ```bash
   cargo fetch
   cargo build --workspace
   cargo test --workspace
   ```

## Development workflow

Detailed build, test, lint, and release-verification commands are in [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md).

Quick reference:

| Purpose | Command |
| --- | --- |
| Build | `cargo build --workspace` |
| Test all | `cargo test --workspace` |
| Test one crate | `cargo test -p genignore-core` |
| Run from source | `cargo run -p genignore-cli -- detect --dry-run` |
| Lint (CI uses) | `cargo clippy --workspace --all-targets -- -D warnings` |
| Format check | `cargo fmt --all -- --check` |
| Release build | `cargo build --release -p genignore-cli` |

## Commit conventions

This project uses [conventional commits](https://www.conventionalcommits.org/) for automated changelog generation via release-please:

- `feat:` new features
- `fix:` bug fixes
- `perf:` performance improvements
- `docs:` documentation changes (hidden from changelog)
- `test:` test changes (hidden from changelog)
- `refactor:` code refactoring (hidden from changelog)
- `chore:` maintenance tasks (hidden from changelog)

Use squash-merge when merging PRs for cleaner changelogs.

## Pull request process

1. Open your PR against `main`.
2. Ensure CI checks pass (`lint-and-test` and `release-validation` jobs in `.github/workflows/ci.yml`).
3. Keep PR scope focused and include or update tests when behavior changes.
4. If changes affect packaging, validate with `cargo build --release -p genignore-cli` and the offline smoke check in `docs/TESTING.md`.

## Testing notes

- Tests use Rust's built-in harness: `#[cfg(test)]` modules plus `crates/<crate>/tests/`.
- `testdata/repos/` contains minimal fixture repos for detector testing &mdash; keep them small and secret-free.
- `testdata/contracts/` contains machine-readable output contracts &mdash; update only for intentional output changes.
- See [`docs/TESTING.md`](docs/TESTING.md) for detailed testing guidance.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
