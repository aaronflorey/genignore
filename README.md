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

Output labels in human-readable mode include `Command:`, `Target:`, `Detected:`, `Final:`, `Added:`, `Included:`, `Excluded:`, `File:`, and `Warning:`.

## License

No `LICENSE` file is currently present in this repository.
