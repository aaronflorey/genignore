<!-- generated-by: gsd-doc-writer -->
# github.com/aaronflorey/genignore

`genignore` is a Go CLI for developers who want to safely generate and maintain only the managed section of a repository `.gitignore` file.

## Installation

```bash
go install github.com/aaronflorey/genignore@latest
```

## Quick start

1. Clone the repository and enter it:

```bash
git clone https://github.com/aaronflorey/genignore.git
cd genignore
```

2. Run detection to create or update the managed `.gitignore` block:

```bash
go run . detect
```

3. Preview changes without writing files:

```bash
go run . detect --dry-run
```

4. Add explicit providers to the managed set:

```bash
go run . add go node
```

## Usage examples

List supported providers:

```bash
genignore list
```

Search provider keys:

```bash
genignore search go
```

Get machine-readable output for scripts/CI:

```bash
genignore detect --json
```

Common commands:

- `genignore detect` — detect providers and rebuild the managed block.
- `genignore add <keys...>` — add providers to the existing managed set.
- `genignore list` — print all supported provider keys in sorted order.
- `genignore search <term>` — filter supported providers by a search term.

Bundled custom providers:

- `ai-agents`
- `wrangler`

## License

License is not specified in this repository (no `LICENSE` file detected).
