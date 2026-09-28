# Testing

## Test framework and setup

This repository uses Rust's built-in test harness. The workspace requires a current stable Rust toolchain (see `rust-toolchain`/mise or `Cargo.toml`), plus `git` for the fixture helpers that initialize temporary repositories. Initialize the template submodule before running tests — `genignore-core/build.rs` embeds its snapshot and fails the build when it is missing:

```bash
git submodule update --init --recursive
cargo fetch
```

## Running tests

Run the full suite:

```bash
cargo test --workspace
```

This command already covers the embedded-asset checks:

- `genignore-core`'s build script embeds the `github-gitignore` snapshot, so `cargo test --workspace` fails when the checked-out submodule files are missing.
- `crates/genignore-detection/tests/` verifies the embedded `rules.json` catalog loads and rejects invalid catalog structure, unsafe rule paths, and duplicate providers.
- `crates/genignore-core/tests/contracts.rs` exercises the byte-exact CLI JSON and managed-block contracts under `testdata/contracts/`.

Run one crate:

```bash
cargo test -p genignore-core
```

Run one test by name:

```bash
cargo test -p genignore-core --test contracts detect_diff_next_vscode_app_contract
```

Run with lint gates equivalent to CI:

```bash
cargo fmt --all -- --check
cargo clippy --workspace --all-targets -- -D warnings
```

Watch mode is not configured in this repository.

## Fixture repositories

Fixture repositories under `testdata/repos/` stay minimal — only detector-relevant files, no history, no vendored code. Tests that assert Git working-tree behavior (`detect`, `add`, `resolve`, `doctor` must run at the repository top-level) copy the fixture to a temporary directory and `git init` it before asserting, so fixture setup stays reproducible and outside-marker assertions never depend on this repository's own `.git`.

## Manual offline release verification

Use this check when validating a release artifact or a local build against the embedded, no-network success proof.

1. Build the binary:

```bash
cargo build --release -p genignore-cli
binary_path="$PWD/target/release/genignore"
```

2. Create a clean temporary git repo and clean home/cache roots with no warm-up state:

```bash
tmp_root="$(mktemp -d)"
repo_dir="$tmp_root/repo"
home_dir="$tmp_root/home"
mkdir -p "$repo_dir" "$home_dir/.cache"
git -C "$repo_dir" init -q
printf '{"name":"offline-check"}\n' > "$repo_dir/package.json"
```

3. Disable network access for the shell you use to run the binary (for example, airplane mode, a network-disabled VM/container, or Linux tools such as `bwrap --unshare-net` or `unshare -n`), then run generation with the clean HOME and cache paths:

```bash
(cd "$repo_dir" && HOME="$home_dir" XDG_CACHE_HOME="$home_dir/.cache" "$binary_path" detect --exclude linux,macos,windows)
```

4. Confirm the manual proof:

```bash
test -f "$repo_dir/.gitignore"
grep -q '# Provenance: github/gitignore@' "$repo_dir/.gitignore"
grep -q 'node_modules/' "$repo_dir/.gitignore"
test -z "$(ls -A "$home_dir/.cache")"
```

Expected result: the command succeeds with network disabled, writes a managed block for the detected `node` provider, and leaves the clean cache directory empty.

## Writing new tests

- Unit tests live in `#[cfg(test)]` modules next to implementation; crate-level behavioral tests live in `crates/<crate>/tests/`.
- Prefer table-driven loops over repeated cases (for example in `crates/genignore-detection/tests/detection.rs`).
- Contract fixtures belong under `testdata/contracts/` and `testdata/repos/`; update them only with intentional, reviewed output changes.
- Prefer embedded-fixture tests for provider catalogs, template content, and JSON rule-catalog behavior so network assumptions do not leak back into the suite.

## Coverage requirements

No explicit coverage threshold is configured in this repository.

| Type | Threshold |
| --- | --- |
| Lines | Not configured |
| Branches | Not configured |
| Functions | Not configured |
| Statements | Not configured |

## CI integration

Tests run in GitHub Actions via `.github/workflows/ci.yml`.

- **Workflow:** `ci`
- **Triggers:** `push`, `pull_request`
- **Job:** `lint-and-test`
- **Test command:** `cargo test --workspace`

No extra CI-only asset check is required today: the crate tests above already make `cargo test --workspace` fail when embedded catalog files or the rule catalog are missing or invalid.

The same job runs `cargo fmt --all -- --check` and `cargo clippy --workspace --all-targets -- -D warnings` before the test step.
