<!-- generated-by: gsd-doc-writer -->

# Configuration

## Environment variables

No project-specific environment variables are defined in this repository (no `.env.example` or `.env.sample`, and no direct `os.Getenv`/`LookupEnv` usage in runtime code).

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| _None_ | N/A | N/A | Runtime configuration is loaded from a TOML file in the user home directory. |

## Config file format

`genignore` reads machine-level defaults from:

- `$HOME/.config/genignore/config.toml` (see `internal/app/config.go`, `configRelativePath`)

Supported shape:

- Top-level table: `defaults`
- `defaults.providers` (`[]string`): default provider keys used by `detect` when `--include` is not set
- `defaults.ignore_rules` (`[]string`): extra ignore rules appended into the managed block
- Top-level table: `runtime`
- `runtime.offline` (`bool`): when `true`, skip live GitHub template refreshes and require cached remote template content for remote providers

Minimal working example:

```toml
[defaults]
providers = ["go", "node"]
ignore_rules = [".direnv/", "coverage.out"]

[runtime]
offline = true
```

Validation behavior:

- Unknown fields are rejected (`toml.Decoder.DisallowUnknownFields()` in `LoadConfig`).
- If the file is missing, config loading returns an empty config (not an error).

## Required vs optional settings

All settings are optional.

- Config file path (`$HOME/.config/genignore/config.toml`): optional.
- `defaults.providers`: optional.
- `defaults.ignore_rules`: optional.
- `runtime.offline`: optional.

Startup fails only when:

- resolving the user home directory fails (`os.UserHomeDir` via `userHomeDir`),
- the file exists but is invalid TOML,
- a field has the wrong type (for example, `providers = "go"`), or
- the file cannot be read/stat'ed due to filesystem errors.

## Defaults

When no config file is present, `LoadConfig()` returns the zero-value `Config` (empty provider and ignore-rule lists).

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `defaults.providers` | Optional | `[]` | Used by `Detect` only when `--include` is omitted (`internal/app/service.go`). |
| `defaults.ignore_rules` | Optional | `[]` | Passed into managed-block generation as extra rules (`internal/app/service.go`). |
| `runtime.offline` | Optional | `false` | Reuses cached remote templates and skips live GitHub refreshes for remote providers (`internal/api/client.go`). |

Independent of config file values, managed block normalization always enforces these env rules: `.env`, `.env.*`, `!.env.example`, and `!.env.ci` (`requiredEnvRules` in `internal/gitignore/manager.go`).

## Runtime source behavior

Supported-provider validation comes from the checked-in GitHub catalog snapshot shipped with the binary, plus the embedded `ai-agents` and `wrangler` templates.

- Normal online runs fetch remote template bodies from `github/gitignore` and refresh the local template cache.
- `runtime.offline = true` skips the live GitHub fetch and reuses cached remote template bodies instead.
- Offline runs fail clearly when a required cached remote template is missing.
- Remote upstream drift is surfaced through command warnings instead of silently changing the local supported-provider contract.

## Per-environment overrides

No built-in environment-specific config files are implemented (no `.env.development`, `.env.production`, or `.env.test` handling in runtime code).

To use different settings across environments, provide different home-directory config files at `$HOME/.config/genignore/config.toml` per machine/user context.
