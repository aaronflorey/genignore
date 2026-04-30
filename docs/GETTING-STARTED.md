<!-- generated-by: gsd-doc-writer -->
# Getting Started

## Prerequisites

- `Go >= 1.22` (from `go.mod`)
- `git` (for cloning the repository)
- Internet access to GitHub APIs (`api.github.com` and `raw.githubusercontent.com`) for provider catalog/template fetches

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

2. **Remote provider fetch failures**
   - Symptom: command errors like `request list API` / `list API returned status ...` / `template API returned status ...`.
   - Fix: verify network access to GitHub and retry.

3. **No providers selected**
   - Symptom: `error: no providers selected after include/exclude`.
   - Fix: run from a project directory with detectable files, or explicitly include providers:

   ```bash
   go run . detect --include go --include node
   ```

## Next steps

- See [DEVELOPMENT.md](DEVELOPMENT.md) for local development workflows and command reference.
- See [TESTING.md](TESTING.md) for test commands and CI test behavior.
