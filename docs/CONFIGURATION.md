<!-- generated-by: gsd-doc-writer -->

# Configuration

## Environment variables

`genignore` does not define any project-specific environment variables in the repository, and there is no `.env.example`/`.env.sample` file.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| _None_ | N/A | N/A | Configuration is loaded from a TOML file at `$HOME/.config/genignore/config.toml`. |

## Config file format

The CLI reads machine-level defaults from `~/.config/genignore/config.toml` (`internal/app/config.go`).

- Top-level table: `defaults`
- Supported keys under `defaults`:
  - `providers` (`[]string`): default provider keys used by `detect` and prepended for `add`
  - `ignore_rules` (`[]string`): additional ignore patterns appended to the generated managed block

Minimal example:

```toml
[defaults]
providers = ["go", "node"]
ignore_rules = [".direnv/", "coverage.out"]
```

Notes:

- Unknown fields are rejected (`DisallowUnknownFields` in `LoadConfig`).
- If the config file does not exist, `genignore` continues with an empty config.

## Required vs optional settings

All configuration settings are optional.

- `~/.config/genignore/config.toml`: optional. If missing, `LoadConfig()` returns an empty config.
- `[defaults].providers`: optional list.
- `[defaults].ignore_rules`: optional list.

Failure conditions:

- The command fails if the file exists but contains invalid TOML or wrong field types (for example, `providers = "go"` instead of an array).
- The command also fails if the config file cannot be read/stat'ed due to filesystem errors.

## Defaults

There are no hard-coded provider defaults in source, but managed block generation always includes required environment ignore rules: `.env`, `.env.*`, `!.env.example`, and `!.env.ci` (`internal/gitignore/manager.go`).

| Setting | Default value | Source |
| --- | --- | --- |
| `defaults.providers` | empty list (`[]`) | `LoadConfig()` returns zero-value `Config` when file is absent (`internal/app/config.go`) |
| `defaults.ignore_rules` | empty list (`[]`) | `LoadConfig()` returns zero-value `Config` when file is absent (`internal/app/config.go`) |

Runtime behavior that depends on these defaults:

- `detect`: uses `defaults.providers` only when `--include` is not provided (`internal/app/service.go`).
- `add`: prepends `defaults.providers` before explicit keys (`internal/app/service.go`).
- Managed block generation always includes `defaults.ignore_rules` as extra rule sources (`internal/app/service.go`).

## Per-environment overrides

No built-in per-environment configuration files are implemented (no `.env.development`, `.env.production`, or `.env.test` handling in this repository).

To use different settings per environment, maintain different `config.toml` files per machine/user profile and run `genignore` in that environment with the corresponding home directory.
