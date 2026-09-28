# Getting Started

## Prerequisites

- A current stable Rust toolchain (`cargo`/`rustc`; see `mise.toml` for the pinned version)
- `git` (for cloning the repository)
- No runtime network access is required for normal CLI use; provider catalogs, template bodies, and repository detection rules ship embedded in the binary

## Installation

### Option A: Install the binary (recommended)

```bash
cargo install --git https://github.com/aaronflorey/genignore genignore-cli
```

If you use Homebrew and the tap is available:

```bash
brew install aaronflorey/tap/genignore
```

### Option B: Build from source

1. Clone the repository:

```bash
git clone https://github.com/aaronflorey/genignore.git
cd genignore
```

2. Initialize submodules (required for embedded templates):

```bash
git submodule update --init --recursive
```

3. Build the CLI:

```bash
cargo build --release -p genignore-cli
```

The binary lands at `target/release/genignore`.

## First run

From the top level of a git repository, detect providers and update the managed block in `.gitignore`:

```bash
genignore detect
```

If building from source:

```bash
cargo run -p genignore-cli -- detect
```

`detect`, `add`, `resolve`, and `doctor` must be run from the top-level directory of a Git working tree; `list` and `search` work anywhere.

## Verify the setup

1. Confirm the binary is available:

```bash
genignore list
```

   You should see a list of supported provider keys.

2. Preview detection without writing files:

```bash
genignore detect --dry-run
```

3. Check diagnostics:

```bash
genignore doctor
```

## Common setup issues

1. **`cargo` command not found or wrong version**
   - Symptom: build/run commands fail before execution.
   - Fix: install Rust via `rustup` (or `mise install` to match the pinned version) and verify with:

   ```bash
   cargo --version
   ```

2. **Not at a repository root**
   - Symptom: commands fail with `must run from the top-level of a Git working tree`.
   - Fix: `cd` to the directory `git rev-parse --show-toplevel` prints and re-run.

3. **Config file rejected**
   - Symptom: startup fails with `invalid config file ...`.
   - Fix: remove unsupported or misspelled fields; only the `[defaults]` table with `providers` and `ignore_rules` is accepted. See [Configuration](CONFIGURATION.md).

4. **No providers selected**
   - Symptom: `error: no providers selected after include/exclude`.
   - Fix: run from a project directory with detectable files, or explicitly include providers:

   ```bash
   genignore detect --include go --include node
   ```

5. **Embedded template initialization failure**
   - Symptom: build or startup fails with `error: initialize embedded templates: ...` or a build-script panic about the template catalog.
   - Fix: ensure the `github/gitignore` submodule is initialized:

   ```bash
   git submodule update --init --recursive
   ```

For more issues, see [Troubleshooting](troubleshooting.md).

## Next steps

- [CLI Reference](cli.md) — all commands, flags, and examples.
- [Configuration](CONFIGURATION.md) — machine-level defaults and strict TOML validation.
- [Development](DEVELOPMENT.md) — local development workflows and build commands.
- [Testing](TESTING.md) — test commands and CI test behavior.
