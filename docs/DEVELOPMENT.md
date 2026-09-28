# Development

## Local setup

1. Fork the repository, then clone your fork and enter the project directory:

```bash
git clone <your-fork-url>
cd genignore
```

2. Confirm your Rust toolchain is installed (`rustup` or `mise install` for the pinned version in `mise.toml`):

```bash
cargo --version
```

3. Initialize the template submodule (required — `genignore-core/build.rs` embeds it):

```bash
git submodule update --init --recursive
```

4. Run a local verification pass before opening a PR:

```bash
cargo fmt --all -- --check
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace
cargo build --release -p genignore-cli
```

5. Run commands directly from source while developing:

```bash
cargo run -p genignore-cli -- detect --dry-run
```

## Build commands

This repository does not use npm-based script runners. Development and validation use Cargo commands plus release tooling.

| Command | Description |
| --- | --- |
| `cargo build --workspace` | Compile all crates in this workspace. |
| `cargo test --workspace` | Run all tests in this workspace. |
| `cargo run -p genignore-cli -- detect` | Run provider detection from source and update the managed `.gitignore` block. |
| `cargo run -p genignore-cli -- detect --dry-run` | Preview detection output and file action without writing. |
| `cargo run -p genignore-cli -- add <keys...>` | Add provider keys to the existing managed set. |
| `cargo run -p genignore-cli -- list` | Print all supported provider keys from the provider catalog. |
| `cargo run -p genignore-cli -- search <term>` | Search provider keys by term. |
| `cargo build --release -p genignore-cli` | Build the release binary (equivalent intent to CI release validation). |

## Code style

- **Formatting:** use `cargo fmt --all` for changed files; CI checks with `cargo fmt --all -- --check`.
- **Linting tool:** CI runs `cargo clippy --workspace --all-targets -- -D warnings` in `.github/workflows/ci.yml` (`lint-and-test` job).
- **Lint configuration:** clippy is configured through crate-level lints in `Cargo.toml` rather than a separate config file.
- **CI quality gate:** the same CI job also runs `cargo test --workspace`.
- **Hooks:** `hk` runs the pre-commit/CI linters defined in `hk.pkl` (`cargo fmt`, `cargo clippy`, secrets, whitespace, etc.).

## Branch conventions

- The repository's release workflow runs on pushes to `main` (`.github/workflows/release.yaml`), so `main` is the effective default branch.
- No repository-specific branch naming convention is documented.

## PR process

- Open your PR against `main`.
- Ensure GitHub Actions checks pass in `.github/workflows/ci.yml` (`lint-and-test` and `release-validation`).
- Use conventional commit types (`feat`, `fix`, `perf`, `docs`, `test`, `refactor`, `chore`) so `release-please` can map commit types to changelog sections (`release-please-config.json`).
- Keep PR scope focused and include or update tests when behavior changes.
- If changes affect packaging, validate the release build locally with `cargo build --release -p genignore-cli` and run the offline smoke check in `docs/TESTING.md`.
