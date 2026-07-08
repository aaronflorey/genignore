<!-- generated-by: gsd-doc-writer -->
# Getting Started

## Prerequisites

- `Go >= 1.22` (from `go.mod`)
- `git` (for cloning the repository)
- No runtime network access is required for normal CLI use; provider catalogs, template bodies, and repository detection rules ship embedded in the binary

## Installation steps

1. Clone the repository:

```bash
git clone git@github.com:aaronflorey/genignore.git
```

2. Enter the project directory:

```bash
cd genignore
```

3. Build the CLI locally:

```bash
go build ./...
```

## First run

Run provider detection and update the managed block in `.gitignore`:

```bash
go run . detect
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
   - Fix: remove unsupported or misspelled fields; only the `[defaults]` table with `providers` and `ignore_rules` is accepted.

3. **No providers selected**
   - Symptom: `error: no providers selected after include/exclude`.
   - Fix: run from a project directory with detectable files, or explicitly include providers:

   ```bash
   go run . detect --include go --include node
   ```

## Next steps

- See [DEVELOPMENT.md](DEVELOPMENT.md) for local development workflows and command reference.
- See [CONFIGURATION.md](CONFIGURATION.md) for machine-level defaults and strict TOML validation.
- See [TESTING.md](TESTING.md) for test commands and CI test behavior.
