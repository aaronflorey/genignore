# Getting Started

## Prerequisites

- `Go >= 1.22` (from `go.mod`)
- `git` (for cloning the repository)
- No runtime network access is required for normal CLI use; provider catalogs, template bodies, and repository detection rules ship embedded in the binary

## Installation

### Option A: Install the binary (recommended)

```bash
go install github.com/aaronflorey/genignore@latest
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
go build ./...
```

## First run

From any project directory, detect providers and update the managed block in `.gitignore`:

```bash
genignore detect
```

If building from source:

```bash
go run . detect
```

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

1. **`go` command not found or wrong version**
   - Symptom: build/run commands fail before execution.
   - Fix: install Go 1.22+ and verify with:

   ```bash
   go version
   ```

2. **Config file rejected**
   - Symptom: startup fails with `invalid config file ...`.
   - Fix: remove unsupported or misspelled fields; only the `[defaults]` table with `providers` and `ignore_rules` is accepted. See [Configuration](CONFIGURATION.md).

3. **No providers selected**
   - Symptom: `error: no providers selected after include/exclude`.
   - Fix: run from a project directory with detectable files, or explicitly include providers:

   ```bash
   genignore detect --include go --include node
   ```

4. **Embedded template initialization failure**
   - Symptom: startup fails with `error: initialize embedded templates: ...`.
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
