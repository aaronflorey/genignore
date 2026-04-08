# Requirements: gitignore-gen

**Defined:** 2026-04-07
**Core Value:** Generate and maintain a deterministic, safe managed `.gitignore` block that users can run repeatedly without losing their own file content.

## v1 Requirements

Requirements for initial release. Each maps to exactly one roadmap phase.

### Commands

- [x] **CMD-01**: User can run `gitignore-gen detect` to auto-detect providers from project and environment signals
- [x] **CMD-02**: User can run `gitignore-gen detect --include <keys>` to force-add valid providers before generation
- [x] **CMD-03**: User can run `gitignore-gen detect --exclude <keys>` to remove providers before generation
- [x] **CMD-04**: User sees a hard error when `detect` resolves to an empty final provider set
- [x] **CMD-05**: User can run `gitignore-gen add <keys...>` to append only missing valid providers to the managed set
- [x] **CMD-06**: User can discover available provider keys using list/search command(s)

### Detection

- [x] **DET-01**: User gets provider detection based on project files/folders, runtime OS, and installed software signals
- [x] **DET-02**: User can inspect detection evidence (`reason`, `source`, `path` when available) in verbose or JSON output
- [x] **DET-03**: User receives warnings for invalid/unsupported provider keys while valid keys still continue
- [x] **DET-04**: User gets deterministic provider ordering (alphabetical) for file metadata, API requests, and outputs

### Gitignore Management

- [x] **GIT-01**: User gets a new `.gitignore` with a managed block when the file does not exist
- [x] **GIT-02**: User keeps all original `.gitignore` content unchanged outside managed markers when file exists without markers
- [x] **GIT-03**: User keeps all content before and after markers unchanged when markers already exist; only managed region is replaced
- [x] **GIT-04**: User edits inside managed markers are overwritten on regeneration
- [x] **GIT-05**: User sees only deterministic block updates between repeated equivalent runs (idempotent behavior)

### API Integration

- [x] **API-01**: User gets template generation from Toptal API for the final provider set
- [x] **API-02**: User gets hard-failure behavior when API calls fail
- [x] **API-03**: User receives warning if a hardcoded supported provider is missing from the remote provider list

### Output and Automation

- [x] **OUT-01**: User gets concise human-readable output showing detected/added/excluded providers, warnings, and file action
- [x] **OUT-02**: User can run `--json` to receive machine-readable execution data (command, cwd, selections, warnings, detection reasons, file action)
- [x] **OUT-03**: User can run `--dry-run` to preview actions without writing `.gitignore`

### Testing and Quality

- [x] **TST-01**: User-facing behavior is covered by deterministic fixture-based tests without live API dependency
- [x] **TST-02**: File migration scenarios are tested (missing file, existing file without markers, existing file with markers)
- [x] **TST-03**: Selection semantics are tested (`detect` reset, `add` append-only, ordering stability, invalid-key warnings)
- [x] **TST-04**: Failure-path and output contracts are tested (API failure, dry-run behavior, JSON structure)

## v2 Requirements

Deferred to future release.

### Expansion

- **EXP-01**: User can manage multiple repositories in monorepo mode from one command
- **EXP-02**: User can extend detection/provider support through a plugin system
- **EXP-03**: User can generate from local/offline cached templates when API is unavailable
- **EXP-04**: User can configure default behavior with a project config file

## Out of Scope

| Feature | Reason |
|---------|--------|
| Monorepo handling in v1 | Explicitly excluded by PRD to keep v1 scoped and safe |
| Plugin detector/provider architecture in v1 | Defer until core detection model is validated |
| Offline mode/local fallback in v1 | API failures should be explicit hard failures |
| Persistent cache across executions in v1 | Unnecessary for v1 and adds invalidation complexity |
| Project config file in v1 | Explicitly excluded to keep first release simple |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CMD-01 | Phase 3 | Completed |
| CMD-02 | Phase 3 | Completed |
| CMD-03 | Phase 3 | Completed |
| CMD-04 | Phase 3 | Completed |
| CMD-05 | Phase 2 | Complete |
| CMD-06 | Phase 2 | Complete |
| DET-01 | Phase 3 | Completed |
| DET-02 | Phase 3 | Completed |
| DET-03 | Phase 2 | Complete |
| DET-04 | Phase 2 | Complete |
| GIT-01 | Phase 1 | Complete |
| GIT-02 | Phase 1 | Complete |
| GIT-03 | Phase 1 | Complete |
| GIT-04 | Phase 1 | Complete |
| GIT-05 | Phase 1 | Complete |
| API-01 | Phase 2 | Complete |
| API-02 | Phase 2 | Complete |
| API-03 | Phase 2 | Complete |
| OUT-01 | Phase 4 | Completed |
| OUT-02 | Phase 4 | Completed |
| OUT-03 | Phase 4 | Completed |
| TST-01 | Phase 5 | Complete |
| TST-02 | Phase 5 | Complete |
| TST-03 | Phase 5 | Complete |
| TST-04 | Phase 5 | Complete |

**Coverage:**
- v1 requirements: 25 total
- Mapped to phases: 25
- Unmapped: 0 ✅

---
*Requirements defined: 2026-04-07*
*Last updated: 2026-04-08 after Phase 5 verification*
