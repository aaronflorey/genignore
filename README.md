<!-- generated-by: gsd-doc-writer -->
# genignore

`genignore` is a Go CLI for developers who want deterministic `.gitignore` generation while preserving manual rules outside a managed marker block.

## Installation

```bash
go install github.com/aaronflorey/genignore@latest
```

<!-- VERIFY: Homebrew installation is available once the tap repository is published. -->
If you use Homebrew:

```bash
brew install aaronflorey/tap/genignore
```

## Quick start

1. Detect providers in the current directory and create or update the managed `.gitignore` block:

```bash
genignore detect
```

2. Preview what would change without writing files:

```bash
genignore detect --dry-run
```

3. Add specific providers to the existing managed set:

```bash
genignore add go node
```

4. Reuse previously fetched remote templates without a live GitHub call:

```toml
[runtime]
offline = true
```

If you are working from source:

```bash
git clone https://github.com/aaronflorey/genignore.git
cd genignore
go run . detect
```

## Usage examples

List all supported provider keys:

```bash
genignore list
```

Search providers by term:

```bash
genignore search jetbrains
```

Run detection with machine-readable JSON output:

```bash
genignore detect --json
```

Exclude certain providers from detection:

```bash
genignore detect --exclude windows,macos
```

Enable explicit offline template reuse from the machine-level config file:

```toml
[runtime]
offline = true
```

`runtime.offline = true` keeps provider validation on the checked-in GitHub catalog snapshot and loads remote template content from the local cache created by prior online runs. If a required cached remote template is missing, `genignore` fails clearly instead of silently widening support or falling back to a live refresh.

The canonical supported-provider contract is the checked-in GitHub catalog snapshot shipped with `genignore`, plus the embedded `ai-agents` and `wrangler` exceptions. Live GitHub fetches are still used to download remote template bodies during normal online runs, and any upstream drift is surfaced as warnings instead of changing the local support contract implicitly.

Output labels in human-readable mode include `Command:`, `Target:`, `Detected:`, `Final:`, `Added:`, `Included:`, `Excluded:`, `File:`, and `Warning:`.

Default editor detection is intentionally repo-backed: `visualstudiocode` is detected from `.vscode/` or `*.code-workspace`, and `jetbrains` is detected from `.idea/` or `*.iml`. Installed editors alone do not change default detection results.

## License

No `LICENSE` file is currently present in this repository.
