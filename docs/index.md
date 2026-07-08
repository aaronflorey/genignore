# genignore Documentation

`genignore` is a Go CLI that detects project providers in the current directory, loads embedded `.gitignore` template content from the checked-in `github/gitignore` snapshot plus local custom templates, and updates only the managed marker block in `.gitignore` — preserving all user-owned lines outside the markers.

## What this documentation covers

| Document | Purpose |
| --- | --- |
| [Getting Started](GETTING-STARTED.md) | Prerequisites, installation, first run, and verification. |
| [Configuration](CONFIGURATION.md) | Config file format, environment variables, defaults, validation, and precedence. |
| [Troubleshooting](troubleshooting.md) | Common problems, symptoms, causes, and fixes. |
| [CLI Reference](cli.md) | Commands, flags, arguments, JSON output, and examples. |
| [Architecture](ARCHITECTURE.md) | System overview, component diagram, data flow, and key abstractions. |
| [Development](DEVELOPMENT.md) | Local setup, build commands, code style, and PR process. |
| [Testing](TESTING.md) | Test framework, commands, fixtures, CI integration, and offline verification. |

## Recommended reading order

### First-time users

1. [Getting Started](GETTING-STARTED.md) — install and run your first `detect`.
2. [CLI Reference](cli.md) — learn the available commands and flags.
3. [Configuration](CONFIGURATION.md) — set machine-level defaults if needed.
4. [Troubleshooting](troubleshooting.md) — resolve common issues.

### Maintainers and contributors

1. [Getting Started](GETTING-STARTED.md) — confirm your toolchain works.
2. [Architecture](ARCHITECTURE.md) — understand the package layout and data flow.
3. [Development](DEVELOPMENT.md) — local workflow, build, and PR process.
4. [Testing](TESTING.md) — test commands, fixtures, and CI behavior.
5. [Configuration](CONFIGURATION.md) — config validation and embedded catalog behavior.
6. [CLI Reference](cli.md) — command surface and flag semantics.

## Key concepts

- **Managed block**: the content between `# BEGIN genignore` and `# END genignore` in `.gitignore`. Only this block is owned by the CLI; everything outside is preserved.
- **Provider detection**: the CLI scans the current directory and one level of subdirectories for project signal files (e.g., `go.mod`, `package.json`, `Cargo.toml`) using embedded detection rules from `internal/rulecatalog/rules.json`.
- **Embedded templates**: all provider templates ship embedded in the binary — no network access is required for normal operation.
- **Provenance**: managed blocks include a `# Provenance:` line recording the pinned `github/gitignore` commit and any embedded custom providers that contributed content.

## Known limitations

- **Scope is the current directory only**: no monorepo traversal or recursive project discovery beyond one level of subdirectories.
- **Machine-level config only**: no per-project preset files or repository-local configuration. Only `$HOME/.config/genignore/config.toml` is supported.
- **No plugin system**: providers and detectors are compiled into the binary. Adding a custom provider requires source changes (see [Development](DEVELOPMENT.md)).
- **No `LICENSE` file**: the repository does not currently include a license file.
