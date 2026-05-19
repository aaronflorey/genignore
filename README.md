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

Or preview the exact managed-block diff without writing files:

```bash
genignore detect --diff
```

3. Add specific providers to the existing managed set:

```bash
genignore add go node
```

4. Reuse previously fetched remote templates without a live GitHub call:

```toml
[runtime]
offline = true
upstream_commit = "3780fff86c705155792fb3e1787cebd6281ba8cf"
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

Explain the current detector evidence, provider resolution, cache state, and runtime decisions:

```bash
genignore doctor
genignore doctor --json
```

Exclude certain providers from detection:

```bash
genignore detect --exclude windows,macos
```

Enable explicit offline template reuse from the machine-level config file:

```toml
[runtime]
offline = true
upstream_commit = "3780fff86c705155792fb3e1787cebd6281ba8cf"
```

`runtime.upstream_commit` pins remote catalog lookups and remote template fetches to a specific `github/gitignore` commit so equivalent inputs can regenerate against the same upstream revision later. If omitted, `genignore` uses the checked-in default pin `3780fff86c705155792fb3e1787cebd6281ba8cf`.

`runtime.offline = true` keeps provider validation on the checked-in GitHub catalog snapshot and loads remote template content from the local cache created by prior online runs. Offline cache reuse now requires matching metadata for the pinned upstream commit, a valid integrity checksum, and a fresh fetch timestamp. If a required cached remote template is missing, stale, or corrupt, `genignore` fails clearly instead of silently widening support or falling back to a live refresh.

The canonical supported-provider contract is the checked-in GitHub catalog snapshot shipped with `genignore`, plus the embedded `ai-agents` and `wrangler` exceptions. Live GitHub fetches are still used to download remote template bodies during normal online runs, and any upstream drift is surfaced as warnings instead of changing the local support contract implicitly.

Online runs store cache metadata alongside remote catalog and template bodies, including the pinned upstream commit, `ETag`, fetch time, and checksum. When cached metadata is still valid, `genignore` sends `If-None-Match` and reuses cached content on `304 Not Modified` instead of downloading the same payload again.

Generated managed blocks now include a deterministic `# Provenance:` line that records the pinned `github/gitignore` commit and any embedded providers that contributed content.

`genignore doctor` is the supported diagnostics surface for detector evidence, provider resolution, cache state, and degraded-runtime decisions. Detection entries classify repository-backed evidence separately from host-only heuristics such as runtime OS or installed-application checks.

`genignore detect --diff` and `genignore add --diff` preview the exact managed-block change without writing `.gitignore`. The preview reports the same `File:` action that the eventual write path would take: `created`, `updated`, or `no-op`.

Output labels in human-readable mode include `Command:`, `Target:`, `Detected:`, `Final:`, `Added:`, `Included:`, `Excluded:`, `File:`, `Preview:`, `Diff:`, `Warning:`, `Detection:`, `Offline:`, `Upstream:`, `Remote:`, `Embedded:`, `Cache:`, `Decision:`, and `Provenance:`.

Default editor detection is intentionally repo-backed: `visualstudiocode` is detected from `.vscode/` or `*.code-workspace`, and `jetbrains` is detected from `.idea/` or `*.iml`. Installed editors alone do not change default detection results.

## Development

Run the managed-block regression tests:

```bash
go test ./internal/gitignore
```

Run the managed-block rewrite benchmarks intentionally:

```bash
go test -run '^$' -bench . ./internal/gitignore
```

Run the managed-block fuzz targets intentionally:

```bash
go test -run '^$' -fuzz=FuzzParseManagedProvidersRoundTrip -fuzztime=10s ./internal/gitignore
go test -run '^$' -fuzz=FuzzMergeManagedBlock -fuzztime=10s ./internal/gitignore
```

## Release maintenance

CI validates release packaging with GoReleaser `v2.15.2` by building snapshot archives and unpacking the Linux amd64 tarball before running the packaged `genignore` binary.

When refreshing the toolchain or release dependencies, keep that work isolated from feature changes and verify it deliberately:

1. Update one pinned tool or dependency at a time, such as `go.mod`, `go.sum`, `.github/workflows/*.yml`, or `.goreleaser.yaml`.
2. Run `go test ./...`.
3. Run `goreleaser check`.
4. Run `goreleaser release --snapshot --clean --skip=publish`.
5. Unpack `dist/genignore_*_linux_amd64.tar.gz` and run `./genignore list` or `./genignore help` from the extracted artifact.

Treat upstream catalog snapshot refreshes and Go or GoReleaser version bumps as narrow maintenance changes so any release regression stays reviewable.

## License

No `LICENSE` file is currently present in this repository.
