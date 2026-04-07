# Pitfalls Research

**Domain:** gitignore generation CLI (detect + managed block + remote template API)
**Researched:** 2026-04-07
**Confidence:** MEDIUM-HIGH

## Critical Pitfalls

### Pitfall 1: Destructive `.gitignore` rewrites outside managed markers

**What goes wrong:**
The tool rewrites the full file (or mis-parses marker boundaries), deleting user-owned rules before/after the generated section.

**Why it happens:**
Implementations treat `.gitignore` as tool-owned, not *partially* tool-owned.

**How to avoid:**
- Treat only `# BEGIN gitignore-gen` ... `# END gitignore-gen` as mutable.
- On missing markers: prepend generated block, append original file unchanged.
- Write via temp file + atomic replace (never truncate first, then write).
- Add invariant tests: “bytes before marker unchanged”, “bytes after marker unchanged”.

**Warning signs:**
- User reports manual rules disappearing after `detect`/`add`.
- Diffs show unrelated line movement outside block.
- Empty or 0-byte `.gitignore` after interrupted run.

**Phase to address:**
Phase 1 — File manager + marker parser foundation.

---

### Pitfall 2: Non-atomic writes causing corruption on failure

**What goes wrong:**
Process crash/power loss leaves partial output or empty file.

**Why it happens:**
Naive `os.WriteFile`/truncate flow instead of staged write + rename.

**How to avoid:**
- Stage content in same directory, `fsync`, rename over target.
- Use a tested atomic-write helper (or equivalent internal abstraction).
- Keep backup path in memory for rollback on write failure.

**Warning signs:**
- Rare flaky CI failures around file tests.
- Users see malformed marker blocks.
- Intermittent 0-byte `.gitignore` bug reports.

**Phase to address:**
Phase 1 — Safe write path before detection/API work.

---

### Pitfall 3: Incorrect ignore semantics (negation/order/precedence)

**What goes wrong:**
Generated rules behave differently than expected; tool appears “wrong” even if API data is valid.

**Why it happens:**
Ignoring Git’s precedence and pattern semantics (especially `!` negation and tracked-file behavior).

**How to avoid:**
- Keep API template output as-is; do not reorder lines inside template body.
- Validate tricky examples with `git check-ignore -v` fixtures.
- Document clearly: `.gitignore` does not affect already tracked files.

**Warning signs:**
- “Rule exists but file is still tracked” confusion.
- Support churn on negation behavior.
- Tests only check string equality, not Git behavior.

**Phase to address:**
Phase 2 — Template integration + semantics verification.

---

### Pitfall 4: Over-aggressive detection (false positives) from global software signals

**What goes wrong:**
Providers like IDE/OS are added because software is installed globally, not because project needs those patterns.

**Why it happens:**
No confidence/evidence threshold per detector; “installed anywhere” treated as project intent.

**How to avoid:**
- Require evidence metadata per provider (source + path + reason).
- Gate global detectors with lower priority than project-file detectors.
- In verbose/JSON output, surface *why* each provider was selected.
- Allow clean opt-out with `--exclude` and deterministic reruns.

**Warning signs:**
- Final provider list feels noisy for simple repos.
- Frequent user use of `--exclude` for same providers.
- Detection rationale missing or too vague.

**Phase to address:**
Phase 3 — Detector scoring/tuning + explainability.

---

### Pitfall 5: Provider key drift between hardcoded list and remote API

**What goes wrong:**
CLI accepts keys API no longer serves (or misses new keys); runtime behavior diverges from compile-time assumptions.

**Why it happens:**
Hardcoded provider catalog not reconciled with `/api/list` at runtime.

**How to avoid:**
- Fetch `/api/list` each run (or per process) and compare against local support map.
- Warn on missing remote providers; fail if requested providers cannot be fetched.
- Normalize keys consistently (lowercase, sorted, comma-joined).

**Warning signs:**
- “Unsupported key” warnings for keys that used to work.
- API 404/empty template responses for valid local keys.
- Non-deterministic behavior across days/releases.

**Phase to address:**
Phase 2 — API client + compatibility checks.

---

### Pitfall 6: Broken API request construction (encoding, ordering, and partial failures)

**What goes wrong:**
Combined template fetch fails due to malformed path/query or silently succeeds with incomplete provider set.

**Why it happens:**
Improper URL encoding and delimiter handling for multi-provider requests; swallowing upstream errors.

**How to avoid:**
- URL-encode provider keys and join with commas exactly once.
- Keep alphabetical sorting before request construction.
- Treat API errors/non-200 responses as hard failures (no silent fallback in v1).
- Add contract tests for special keys (e.g., `c++`, `jetbrains+iml`).

**Warning signs:**
- Failures only for certain keys with symbols.
- Generated block missing expected sections without explicit error.
- Human output says “updated” despite API warnings/errors.

**Phase to address:**
Phase 2 — API client hardening.

---

### Pitfall 7: Unscriptable CLI output and weak failure signaling

**What goes wrong:**
Automation cannot rely on output; parse breaks when message text changes; errors mixed into machine output.

**Why it happens:**
No strict contract for `--json`, stdout/stderr mixing, and inconsistent exit codes.

**How to avoid:**
- Emit JSON only to stdout when `--json`.
- Send warnings/errors to stderr in human mode.
- Guarantee stable JSON schema fields for `detect` and `add`.
- Map failure modes to consistent non-zero exits.

**Warning signs:**
- CI scripts using fragile grep on human text.
- Inconsistent exit codes for same error class.
- `--json` output includes prose/debug lines.

**Phase to address:**
Phase 4 — UX/output contract + CLI ergonomics.

---

### Pitfall 8: Test suites coupled to live API/network and happy-path-only fixtures

**What goes wrong:**
Flaky tests, false confidence, and regressions in marker migration/error paths.

**Why it happens:**
Tests hit live Toptal endpoints and don’t model failures, permission issues, or malformed marker cases.

**How to avoid:**
- Use `httptest` server for API simulation (200/4xx/5xx/timeouts).
- Use fixture snapshots for template/list responses.
- Table-test migration states: no file, no markers, valid markers, broken markers.
- Add golden tests for JSON schema and deterministic ordering.

**Warning signs:**
- CI failures due to network only.
- Low coverage on failure branches.
- Refactors repeatedly break output shape/ordering.

**Phase to address:**
Phase 5 — reliability and regression safety net.

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Full-file rewrite instead of managed-block replacement | Fast initial implementation | Data-loss risk, user trust damage | Never |
| Live API in unit tests | No fixture maintenance | Flaky CI, nondeterministic tests | Never |
| Detection without evidence metadata | Faster detector code | Impossible to debug false positives | Never |
| No JSON schema contract | Quicker CLI iteration | Automation breakage with minor wording changes | Only for pre-alpha spikes |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Toptal API list endpoint | Assuming local provider map is always current | Compare local map against `/api/list` each run; warn on drift |
| Toptal combined template endpoint | Joining keys without robust encoding/sorting | Normalize + sort + encode keys before building request |
| Git behavior validation | Assuming textual pattern presence equals effective ignore | Use `git check-ignore -v` behavior-based tests |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Re-fetch list/templates repeatedly in one command | Slow command in large detector set | Cache API responses in-process for run duration | Visible even at small scale (local UX) |
| Scanning too many global install paths synchronously | `detect` feels hung | Bound search paths and add detector timeouts | User machines with large disks/network mounts |
| Verbose output with huge per-detector payload by default | Noisy unusable CLI | Keep concise default; detailed info only with `--verbose`/`--json` | Immediately noticeable |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Following symlinks blindly when reading/writing `.gitignore` | Writing outside intended repo path | Resolve and validate target path is within cwd repo boundary |
| Emitting absolute local paths in default output | Leaks workstation details in logs/screenshots | Keep sensitive paths in verbose mode only |
| Treating API text as trusted metadata/comments | Unexpected output injection into managed block comments | Sanitize metadata comments; keep template body raw but wrapped in controlled markers |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| `detect` silently removing prior manual additions | Surprise and distrust | Make reset semantics explicit in command help and output summary |
| `--dry-run` not showing exact diff intent | Users avoid command in real repos | Show actionable preview: providers before/after + file action |
| Generic “something failed” API errors | Hard to recover | Include status code, endpoint, and retry guidance |

## "Looks Done But Isn't" Checklist

- [ ] **Managed block replacement:** Verify bytes outside markers are bit-for-bit unchanged.
- [ ] **API client:** Verify symbol-containing keys (`c++`, `jetbrains+iml`) round-trip correctly.
- [ ] **Detection:** Verify each selected provider includes machine-readable evidence metadata.
- [ ] **JSON mode:** Verify stable schema and no human prose on stdout.
- [ ] **Error handling:** Verify non-zero exit codes for API/write/empty-result failures.
- [ ] **Dry run:** Verify no file writes occur and output still describes exact intended mutation.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Lost user rules due to bad rewrite | HIGH | Restore from git history/backups; patch parser; add invariant tests before re-release |
| Corrupted file from interrupted write | MEDIUM | Re-run with atomic writer fix; reconstruct from previous commit if needed |
| Over-detected noisy providers | LOW | Re-run with `--exclude`; tune detector weighting and release patch |
| API drift breakage | MEDIUM | Refresh support map, add drift warning path, ship hotfix |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Destructive rewrites outside markers | Phase 1 | Fixture test proves pre/post regions unchanged |
| Non-atomic write corruption | Phase 1 | Kill-process test never leaves partial/empty file |
| Ignore semantics mismatch | Phase 2 | `git check-ignore -v` assertions pass for tricky patterns |
| Detection false positives | Phase 3 | Verbose evidence + reduced recurring excludes in tests |
| API drift and bad request building | Phase 2 | Contract tests for list/fetch and symbol keys |
| Unscriptable output | Phase 4 | JSON snapshot tests + stderr/stdout separation tests |
| Flaky network-dependent tests | Phase 5 | CI runs offline against fixtures/httptest only |

## Sources

- Git `gitignore` manual (precedence, negation, tracked-file caveat): https://git-scm.com/docs/gitignore  
  **Confidence:** HIGH
- Git `git-check-ignore` manual (behavior verification tooling): https://git-scm.com/docs/git-check-ignore  
  **Confidence:** HIGH
- gitignore.io / Toptal API docs (list endpoint, API usage patterns, comma-joined templates): https://docs.gitignore.io/use/api.md, https://docs.gitignore.io/use/command-line.md, https://docs.gitignore.io/install/command-line.md  
  **Confidence:** MEDIUM-HIGH
- GitHub `github/gitignore` README (scope separation: project vs global templates): https://github.com/github/gitignore/blob/main/README.md  
  **Confidence:** HIGH
- Go stdlib docs for testing and HTTP stubbing: https://pkg.go.dev/net/http/httptest, https://pkg.go.dev/testing/fstest  
  **Confidence:** HIGH
- Go `os.Rename` caveat (non-atomic on non-Unix platforms) and atomic write package notes: https://pkg.go.dev/os#Rename, https://pkg.go.dev/github.com/google/renameio, https://pkg.go.dev/github.com/creachadair/atomicfile  
  **Confidence:** MEDIUM-HIGH
- CLI UX conventions for stdout/stderr, exit codes, and machine-readable output: https://clig.dev/  
  **Confidence:** MEDIUM

---
*Pitfalls research for: gitignore generation CLI*
*Researched: 2026-04-07*
