<!-- generated-by: gsd-doc-writer -->
# Getting Started

## Prerequisites

- `Go >= 1.22` (from `go.mod`)
- `git` (to clone the repository)
- Network access to GitHub APIs at runtime (`api.github.com` and `raw.githubusercontent.com`) for provider/template fetches

## Installation steps

1. Clone the repository:

```bash
git clone git@github.com:aaronflorey/genignore.git
```

2. Enter the project directory:

```bash
cd genignore
```

3. Build the CLI:

```bash
go build ./...
```

## First run

Run detection to generate or update the managed `.gitignore` block in the current directory:

```bash
go run . detect
```

If you want to preview changes without writing files, use:

```bash
go run . detect --dry-run
```

## Common setup issues

1. **Go version too old**
   - Symptom: build or run fails because the toolchain does not satisfy module requirements.
   - Fix: upgrade to `Go >= 1.22` and re-run `go build ./...`.

2. **Network/API access blocked**
   - Symptom: `detect`, `list`, or `search` fails with request errors when fetching provider data/templates.
   - Fix: ensure outbound access to GitHub endpoints used by the CLI (`api.github.com`, `raw.githubusercontent.com`).

3. **Unexpected config parsing error**
   - Symptom: startup fails with an invalid config file error.
   - Fix: validate `$HOME/.config/genignore/config.toml` against the expected keys (`[defaults]`, `providers`, `ignore_rules`) or remove unknown fields.

## Next steps

- See [ARCHITECTURE.md](ARCHITECTURE.md) for system structure and component flow.
- See [CONFIGURATION.md](CONFIGURATION.md) for config and defaults.
- See [DEVELOPMENT.md](DEVELOPMENT.md) and [TESTING.md](TESTING.md) for local development and test workflows.
