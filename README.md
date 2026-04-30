<!-- generated-by: gsd-doc-writer -->
# genignore

`genignore` is a Go CLI for developers who want deterministic `.gitignore` generation while preserving manual rules outside a managed marker block.

## Installation

```bash
go install github.com/aaronflorey/genignore@latest
```

## Quick start

1. Clone and enter the repository:

```bash
git clone https://github.com/aaronflorey/genignore.git
cd genignore
```

2. Detect providers and create/update the managed `.gitignore` block:

```bash
go run . detect
```

3. Preview changes without writing files:

```bash
go run . detect --dry-run
```

4. Add specific providers to the existing managed set:

```bash
go run . add go node
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

Run detection with machine-readable output:

```bash
genignore detect --json
```

Example result labels in human-readable mode include `Detected:`, `Final:`, and `File:`.

## License

No `LICENSE` file is currently present in this repository.
