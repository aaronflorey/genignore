# CLI Reference

`genignore` is a Cobra-based CLI with six commands and two persistent flags. All commands operate on the current working directory only.

## Global flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--json` | bool | `false` | Output machine-readable JSON instead of human-readable text. |
| `--verbose` | bool | `false` | Show verbose detection info (per-detector match status, reason, evidence). |

These flags are available on all commands that produce detection output (`detect`, `resolve`, `add`). The `list` and `search` commands support `--json` but not `--verbose`. The `doctor` command supports `--json` but not `--verbose`.

## Commands

### `genignore detect`

Detects providers in the current directory and creates or updates the managed block in `.gitignore`.

```bash
genignore detect [flags]
```

**Flags**:

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--include` | string slice | none | Provider keys to include in addition to detected providers. Repeatable or comma-separated. |
| `--exclude` | string slice | none | Provider keys to exclude from the final set. Repeatable or comma-separated. |
| `--dry-run` | bool | `false` | Show what would change without writing files. |
| `--diff` | bool | `false` | Show the exact managed-block diff without writing files. |
| `--verbose` | bool | `false` | Show per-detector detection details. |

**Behavior**:

1. Scans the current directory and one level of subdirectories for provider signals.
2. Merges detected providers with `--include` keys, then removes `--exclude` keys.
3. If no `--include` is provided, falls back to `defaults.providers` from config.
4. Loads embedded templates for the final provider set.
5. Builds a normalized managed block and upserts it into `.gitignore`.

**File actions** (reported as `File:` in output):

| Action | Meaning |
| --- | --- |
| `created` | `.gitignore` did not exist; it was created. |
| `updated` | `.gitignore` existed; the managed block was replaced. |
| `no-op` | `.gitignore` existed and the managed block is already current. |
| `dry-run` | `--dry-run` was set; no file was written. |

**Examples**:

```bash
# Detect and update .gitignore
genignore detect

# Preview without writing
genignore detect --dry-run

# Preview the exact diff
genignore detect --diff

# Include specific providers
genignore detect --include go --include node

# Exclude OS providers
genignore detect --exclude linux,macos,windows

# JSON output for scripting
genignore detect --json

# Verbose detection details
genignore detect --verbose
```

### `genignore add <keys...>`

Adds specific provider keys to the existing managed set in `.gitignore`. Requires at least one key argument.

```bash
genignore add <keys...> [flags]
```

**Arguments**: One or more provider keys (e.g., `go node python`).

**Flags**:

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--dry-run` | bool | `false` | Show what would change without writing files. |
| `--diff` | bool | `false` | Show the exact managed-block diff without writing files. |
| `--verbose` | bool | `false` | Show per-detector detection details. |

**Behavior**:

1. Reads existing managed providers from `.gitignore` (between markers).
2. Merges them with the provided keys and `defaults.providers` from config.
3. Filters to supported keys only; unsupported keys produce warnings.
4. Rebuilds and upserts the managed block.

**Examples**:

```bash
# Add providers
genignore add go node

# Preview the addition
genignore add go node --dry-run

# Preview the diff
genignore add python --diff
```

### `genignore resolve`

Resolves detected and explicitly included providers without mutating `.gitignore`. This is the read-only automation surface.

```bash
genignore resolve [flags]
```

**Flags**:

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--include` | string slice | none | Provider keys to include. |
| `--exclude` | string slice | none | Provider keys to exclude. |
| `--verbose` | bool | `false` | Show per-detector detection details. |

**Examples**:

```bash
# Resolve detected providers
genignore resolve

# Include and exclude with JSON output
genignore resolve --include macos --exclude windows --json
```

### `genignore doctor`

Explains the current detector evidence, provider resolution, embedded catalog counts, JSON rule-catalog status, retained embedded custom providers, and managed-block provenance.

```bash
genignore doctor [flags]
```

**Flags**:

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--include` | string slice | none | Provider keys to include. |
| `--exclude` | string slice | none | Provider keys to exclude. |

**Output labels** (human-readable mode):

| Label | Description |
| --- | --- |
| `Command:` | Always `doctor`. |
| `Detected:` | Providers detected in the current directory. |
| `Included:` | Providers explicitly included via `--include` or config defaults. |
| `Excluded:` | Providers explicitly excluded via `--exclude`. |
| `Final:` | The final resolved provider set (sorted). |
| `Warning:` | Unsupported provider key warnings. |
| `Detection:` | Per-detector result: key, origin, status, reason, evidence. |
| `Embedded catalog providers:` | Count of providers in the embedded `github/gitignore` snapshot. |
| `Selected providers:` | The selected provider set. |
| `Rule catalog:` | Status (`loaded` or error) and provider count of the embedded JSON rule catalog. |
| `Retained custom providers:` | Embedded custom providers (e.g., `ai-agents`, `wrangler`) in the selection. |
| `Decision:` | Human-readable explanation of runtime decisions. |
| `Provenance:` | The `# Provenance:` line that would be written to the managed block. |

**Detection origins**:

| Origin | Meaning |
| --- | --- |
| `repository` | Evidence comes from files in the project directory. |
| `host` | Evidence comes from the runtime OS or installed applications. |
| `repository+host` | JetBrains install detection combining repository signals with host application checks. |

### `genignore list`

Lists all supported provider keys from the embedded catalog.

```bash
genignore list [flags]
```

**Flags**: `--json` (global only).

**Example**:

```bash
genignore list
genignore list --json
```

### `genignore search <term>`

Searches supported provider keys by substring (case-insensitive).

```bash
genignore search <term>
```

**Arguments**: Exactly one search term.

**Example**:

```bash
genignore search jetbrains
genignore search go --json
```

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Command succeeded. |
| `1` | Command failed (config error, detection error, file error, or invalid arguments). |

## JSON output

When `--json` is set, all commands output a single JSON object to stdout. The structure varies by command — see `internal/app/types.go` for the exact struct definitions (`CommandResult`, `ResolveResult`, `DoctorResult`, `CatalogResult`).

Key JSON fields across commands:

| Field | Type | Present in | Description |
| --- | --- | --- | --- |
| `command` | string | all | The command name. |
| `cwd` | string | detect, resolve, doctor | The working directory. |
| `detectedProviders` | []string | detect, resolve, doctor | Providers detected by detectors. |
| `includedProviders` | []string | detect, resolve, doctor | Providers from `--include` or config defaults. |
| `excludedProviders` | []string | detect, resolve, doctor | Providers from `--exclude`. |
| `finalProviders` | []string | detect, resolve, doctor | The final resolved provider set. |
| `unsupportedKeyWarnings` | []string | detect, resolve, doctor | Warnings for unrecognized provider keys. |
| `detectionResults` | []object | detect, resolve (verbose) | Per-detector results with key, matched, reason, evidence, error. |
| `fileAction` | string | detect, add | `created`, `updated`, `no-op`, or `dry-run`. |
| `previewOnly` | bool | detect, add | True when `--diff` was used. |
| `diff` | string | detect, add | The managed-block diff (when `--diff` is used). |
| `addedProviders` | []string | add | Providers newly added to the managed set. |
| `providers` | []string | list, search | Matching provider keys. |

## Managed block format

The managed block written to `.gitignore` has this structure:

```text
# BEGIN genignore
# Generated by genignore. Do not edit between these markers.
# Providers: go,node
# Provenance: github/gitignore@<commit> [go,node]
<template content>
<extra ignore rules from config>
.env
.env.*
!.env.example
!.env.ci
# END genignore
```

- The `# Providers:` line records the provider keys as a comma-separated list.
- The `# Provenance:` line records the pinned `github/gitignore` commit hash and the providers sourced from it, plus any embedded custom providers.
- Required env rules (`.env`, `.env.*`, `!.env.example`, `!.env.ci`) are always appended at the end of the block, normalized by `internal/gitignore/manager.go`.
- Historical source comments from older `toptal.com` or `raw.githubusercontent.com` integrations are stripped on rerun.
