# Troubleshooting

This page covers common problems, their symptoms, likely causes, and fixes.

## Startup failures

### `error: failed to get working directory`

**Symptom**: CLI exits immediately with this message.

**Cause**: `os.Getwd()` fails, typically because the current directory was deleted or renamed while the shell was still in it.

**Fix**: Change to a valid directory and re-run.

### `error: resolve config home: ...`

**Symptom**: CLI exits before any command runs.

**Cause**: `os.UserHomeDir()` failed. This can happen when the `HOME` environment variable is unset or empty.

**Fix**: Ensure `HOME` is set:

```bash
echo "$HOME"
```

If empty, set it to your home directory and re-run.

### `error: invalid config file ...`

**Symptom**: CLI exits before any command runs.

**Cause**: The config file at `$HOME/.config/genignore/config.toml` exists but contains invalid TOML, unknown fields, or wrong types.

**Fix**: The config file uses strict TOML decoding (`DisallowUnknownFields`). Only the `[defaults]` table with `providers` and `ignore_rules` is accepted. Remove any unsupported tables (e.g., stale `[runtime]` sections) or misspelled fields.

Valid shape:

```toml
[defaults]
providers = ["go", "node"]
ignore_rules = [".direnv/", "coverage.out"]
```

See [Configuration](CONFIGURATION.md) for full details.

### `error: initialize embedded templates: ...` or `error: initialize provider registry: ...`

**Symptom**: CLI exits before any command runs.

**Cause**: An embedded catalog failed to initialize at startup. This typically means the `github/gitignore` submodule is not checked out or is out of sync.

**Fix**: Initialize and update the submodule:

```bash
git submodule update --init --recursive
```

Then rebuild:

```bash
go build ./...
```

CI always checks out submodules recursively (`.github/workflows/ci.yml`, `submodules: recursive`).

## Detection problems

### `error: no providers selected after include/exclude`

**Symptom**: `detect` or `resolve` fails with this message.

**Cause**: No providers were detected in the current directory, no defaults are configured, and no `--include` keys were provided. Alternatively, all detected providers were excluded.

**Fix**: Either run from a directory with detectable project files, or explicitly include providers:

```bash
genignore detect --include go --include node
```

Or configure machine-level defaults in `$HOME/.config/genignore/config.toml`:

```toml
[defaults]
providers = ["go", "node"]
```

### `Warning: unsupported provider key: <key>`

**Symptom**: A warning line appears in output when using `--include` or `--add` with a key that is not in the embedded provider catalog.

**Cause**: The provider key is not recognized. Supported keys come from the embedded `github/gitignore` snapshot plus the embedded `ai-agents` and `wrangler` custom templates.

**Fix**: Check the list of supported keys:

```bash
genignore list
```

Or search for a specific term:

```bash
genignore search go
```

### Detected providers differ from expectations

**Symptom**: `detect` finds more or fewer providers than expected.

**Diagnosis**: Run `doctor` to see full detection evidence:

```bash
genignore doctor
```

Or with JSON output for scripting:

```bash
genignore doctor --json
```

The `doctor` output shows each detector's match status, origin (`repository`, `host`, or `repository+host`), reason, and evidence path. Use `--verbose` on `detect` or `resolve` for similar detail:

```bash
genignore detect --verbose
genignore resolve --verbose
```

### OS providers (`macos`, `linux`, `windows`) always detected

**Symptom**: The current OS is always detected as a provider.

**Cause**: OS detectors match based on `runtime.GOOS` and are host-only heuristics. They do not inspect repository files.

**Fix**: Exclude them if not needed:

```bash
genignore detect --exclude linux,macos,windows
```

## Managed block issues

### `error: malformed managed markers in .gitignore`

**Symptom**: `detect` or `add` fails with this message.

**Cause**: The `.gitignore` file contains `# BEGIN genignore` or `# END genignore` markers, but they are malformed — either duplicated, out of order, or only one of the pair is present.

**Fix**: Ensure exactly one `# BEGIN genignore` and one `# END genignore` exist, with `BEGIN` before `END`. Remove any duplicate or orphaned markers. If you need to start fresh, remove both markers and the content between them, then re-run `genignore detect`.

### Managed block not updating

**Symptom**: `detect` reports `File: no-op` even when providers changed.

**Cause**: The generated managed block is identical to the existing one. This is expected behavior when the provider set and template content have not changed.

**Diagnosis**: Preview the diff to confirm:

```bash
genignore detect --diff
```

If the diff is empty, the block is already up to date.

### User content outside markers is modified

**Symptom**: Lines outside `# BEGIN genignore` / `# END genignore` appear changed after running `detect`.

**Cause**: This should not happen. The merge logic in `internal/gitignore/manager.go` (`mergeManagedBlock`) only replaces content between the markers. If the file has no markers, the managed block is prepended.

**Fix**: If this occurs, it is a bug. Report it with the exact `.gitignore` content before and after, and the `genignore doctor` output.

## Build and test failures

### Template snapshot tests fail

**Symptom**: `go test ./internal/templatecatalog` fails.

**Cause**: The `github/gitignore` submodule is not initialized or is out of sync with the embedded snapshot.

**Fix**:

```bash
git submodule update --init --recursive
go test ./internal/templatecatalog
```

### Rule catalog tests fail

**Symptom**: `go test ./internal/rulecatalog` fails.

**Cause**: The embedded `rules.json` is missing, invalid, or references a provider not in the embedded template catalog.

**Fix**: Ensure the submodule is initialized, then check `internal/rulecatalog/rules.json` for structural issues. The catalog uses strict JSON decoding and validates that every provider key exists in the embedded template or custom template catalogs.

### GoReleaser validation fails

**Symptom**: `goreleaser check` reports errors.

**Fix**: Run locally to see details:

```bash
mise x -- goreleaser check
```

Common causes include schema changes in GoReleaser v2 or misconfigured build/archive sections in `.goreleaser.yaml`.

## Escalation

For failures not covered here:

1. Run `genignore doctor --json` and review the full diagnostics output.
2. Check the [Architecture](ARCHITECTURE.md) and [CLI Reference](cli.md) docs for expected behavior.
3. Review the relevant source files referenced in the docs for implementation details.
4. If the issue appears to be a bug, open an issue with the `doctor` output and steps to reproduce.
