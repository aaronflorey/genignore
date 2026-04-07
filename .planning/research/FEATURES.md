# Feature Research

**Domain:** Gitignore generation CLI for developers
**Researched:** 2026-04-07
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Generate `.gitignore` from named templates (single and multiple keys) | Core job-to-be-done for this category (`gitignore.io`, `gibo`) | LOW | Must accept multiple template keys and return merged rules in one run. |
| Template discovery (`list` + `search`) | Users cannot reliably guess template keys; competitors expose discovery commands | LOW | `gibo` and modern tools expose list/search; this reduces support burden and invalid input errors. |
| Safe file update behavior (create if missing, preserve non-managed user content) | Reliability expectation for repeat use in real repos | MEDIUM | For this product, managed markers are the right default because they allow deterministic regeneration without clobbering manual rules. |
| Deterministic output (stable provider ordering) | Prevents noisy diffs and trust erosion; also called out by ecosystem pain points | LOW | Sort provider keys before API calls and before rendering metadata/comments. |
| Clear handling of invalid/unknown template keys | CLI ergonomics baseline | LOW | Warn and continue with valid keys; fail only when final provider set is empty for detect/reset flows. |
| Cross-platform command behavior (macOS/Linux/Windows) | CLI audience is multi-OS; major tools publish multi-platform installs | MEDIUM | Keep paths/newlines/permissions consistent; avoid shell-specific assumptions. |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Automatic stack/environment detection (project + OS + installed tooling) | Removes manual template hunting; faster “just make this repo sane” workflow | HIGH | Similar ideas exist (e.g., `gig autogen`), but robust multi-signal detection with evidence is still uncommon. |
| Explainable detection metadata (`why detected`, evidence path/source) | Builds trust and debuggability; reduces “why did it add X?” frustration | MEDIUM | Strong pair with `--verbose` and `--json` output. |
| Explicit `detect` (reset) vs `add` (append-only) semantics | Reduces accidental churn and makes intent obvious | MEDIUM | Distinguishes this tool from simple “append template text” CLIs. |
| Structured `--json` output for CI/automation | Enables scripting, policy checks, editor integration | MEDIUM | Most lightweight generators are human-output-first; JSON unlocks automation workflows. |
| API drift visibility (warn when hardcoded providers diverge from remote source) | Early warning for stale binaries and template ecosystem changes | MEDIUM | Reliability feature for maintainers and users; not common in simple wrappers. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Full plugin ecosystem for detectors/providers in v1 | “Let everyone add custom providers immediately” | Expands surface area before core semantics stabilize; high support and compatibility cost | Keep detector/provider set internal in v1; add extension points only after usage patterns are validated. |
| Monorepo-wide recursive auto-management in v1 | “Handle all packages automatically” | High risk of unintended writes and poor defaults across heterogeneous repos | Scope to current working directory for v1; design monorepo mode as explicit opt-in later. |
| Offline template cache/sync subsystem in v1 | “Must work without internet” | Cache invalidation and template freshness become a product inside the product | Keep hard-fail on API unavailability in v1; revisit offline mode after core reliability is proven. |
| Interactive-only UX (fzf/wizard as primary path) | “More discoverable and friendly” | Adds dependencies and hurts non-interactive automation use cases | Keep non-interactive commands first-class; optional interactive wrapper can come later. |
| Multi-source template merge engine in v1 (GitHub + Toptal + local precedence) | “Maximum flexibility from day one” | Conflict resolution and provenance complexity can undermine deterministic behavior | Use one authoritative source path in v1 (Toptal API), with clear future extension plan. |

## Feature Dependencies

```text
Template discovery (list/search)
    └──enables──> Valid provider selection
                      └──required for──> Generate from templates

Generate from templates
    └──requires──> Safe file update behavior (managed block)
                      └──enables──> Deterministic repeated runs

Deterministic ordering
    └──enhances──> Safe file update behavior (stable diffs)

Detection metadata model
    └──required for──> Explainable detection output
                      └──required for──> Structured JSON output quality

Automatic detection
    └──requires──> Provider catalog + detector implementations

Detect/Add command semantics
    └──conflicts with──> “Single ambiguous generate command” UX
```

### Dependency Notes

- **Template discovery requires source/provider catalog:** list/search must read the same canonical provider namespace used by generation, or users see keys that cannot be applied.
- **Safe updates require explicit ownership boundaries:** without managed markers (or equivalent), repeat generation can overwrite or duplicate user-managed rules.
- **Explainability requires richer detector results:** boolean-only detectors cannot support trustworthy verbose/JSON output.
- **Automatic detection depends on curated detectors:** detection quality is only as good as detector coverage and evidence quality.
- **Detect vs Add split prevents accidental destructive behavior:** reset and append semantics should never be conflated in one default command.

## MVP Definition

### Launch With (v1)

Minimum viable product — what's needed to validate the concept.

- [x] Generate managed `.gitignore` block from selected templates (deterministic order) — essential core behavior.
- [x] `detect` + `add` command model with include/exclude controls — captures automatic + manual workflows.
- [x] Safe idempotent file management (preserve user content outside markers) — reliability requirement.
- [x] Template discovery and unsupported-key warnings — usability and trust baseline.
- [x] `--dry-run`, `--verbose`, and `--json` output modes — operational confidence and automation support.

### Add After Validation (v1.x)

Features to add once core is working.

- [ ] Optional interactive provider picker — add when enough users request guided UX.
- [ ] Optional “suggested providers” ranking/tuning — add after telemetry or user feedback validates heuristics.

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] Offline/local cache mode with freshness policy — defer due complexity and stale-template risk.
- [ ] Monorepo-aware multi-root management — defer until single-root behavior is proven stable.
- [ ] Pluggable provider/detector SDK — defer until extension use cases repeat.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Generate from templates + deterministic ordering | HIGH | LOW | P1 |
| Safe managed block updates | HIGH | MEDIUM | P1 |
| Template discovery (list/search) | HIGH | LOW | P1 |
| Invalid key warnings + empty-set guardrails | HIGH | LOW | P1 |
| Auto-detection with evidence | HIGH | HIGH | P1 |
| JSON output for automation | MEDIUM | MEDIUM | P1 |
| API drift warning | MEDIUM | MEDIUM | P2 |
| Interactive picker | LOW | MEDIUM | P3 |
| Offline cache mode | MEDIUM | HIGH | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | gibo | gitignore.io CLI docs | Our Approach |
|---------|------|------------------------|--------------|
| Generate from template keys | Yes (`dump`) | Yes (`curl .../api/<keys>`) | Yes, but with managed-block ownership and deterministic regeneration. |
| List/search templates | Yes (`list`, `search`, `update`) | API supports list endpoint | First-class discovery commands; same namespace used by generator. |
| Automatic detection | Not core/default | Not documented as built-in | Core differentiator (`detect`) with explainable evidence. |
| Safe repeated mutation of existing `.gitignore` | Mostly append-oriented patterns | Typically fetch-and-write patterns | Managed region replacement only; preserve user sections outside markers. |
| Machine-readable JSON output | Not primary | Not primary | Built-in `--json` for CI/editor tooling. |

## Sources

- https://docs.gitignore.io/use/api (official API docs for list/fetch behavior) — **HIGH**
- https://docs.gitignore.io/install/command-line (official CLI usage patterns) — **HIGH**
- https://github.com/simonwhitaker/gibo (active, widely used CLI; feature baseline) — **HIGH**
- https://docs.github.com/en/rest/gitignore/gitignore (official GitHub template API behavior) — **HIGH**
- https://raw.githubusercontent.com/github/gitignore/main/README.md (template taxonomy and curation guidance) — **HIGH**
- https://raw.githubusercontent.com/shihanng/gig/master/README.md (offline/autogen patterns; older project) — **MEDIUM**
- https://raw.githubusercontent.com/polliard/gitignore/main/README.md (multi-source and section-marker patterns; newer but lower adoption) — **MEDIUM**

---
*Feature research for: gitignore generation CLI*
*Researched: 2026-04-07*
