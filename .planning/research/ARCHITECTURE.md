# Architecture Research

**Domain:** Go CLI for provider detection + managed `.gitignore` updates
**Researched:** 2026-04-07
**Confidence:** HIGH

## Standard Architecture

### System Overview

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│                             CLI/Transport Layer                             │
├──────────────────────────────────────────────────────────────────────────────┤
│  root command   detect command   add command   flags/arg validation         │
│       │               │              │                 │                     │
└───────┴───────────────┴──────────────┴─────────────────┴─────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Application/Use-Case Layer                         │
├──────────────────────────────────────────────────────────────────────────────┤
│ DetectUseCase  AddUseCase  PlanBuilder  ExecutionResultAssembler            │
│ (orchestration only; no direct fs/http/syscalls)                            │
└──────────────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                               Domain Layer                                  │
├──────────────────────────────────────────────────────────────────────────────┤
│ ProviderKey, DetectionEvidence, Detector interface, SelectionEngine         │
│ ManagedBlock model, marker parser, ordering rules (alphabetical)            │
└──────────────────────────────────────────────────────────────────────────────┘
                 │                         │                         │
                 ▼                         ▼                         ▼
┌──────────────────────────┐  ┌──────────────────────────┐  ┌──────────────────────┐
│ Infrastructure: Filesystem│  │ Infrastructure: API      │  │ Infrastructure: Env  │
│ GitignoreRepository       │  │ TemplateClient (Toptal)  │  │ Probers (OS/PATH/app)│
└──────────────────────────┘  └──────────────────────────┘  └──────────────────────┘
                 │                         │                         │
                 └───────────────┬─────────┴───────────────┬─────────┘
                                 ▼                         ▼
                         Output/Presenter Layer      JSON/Human Renderers
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| `cmd/*` command handlers | Parse args/flags, validate CLI contract, call use-cases | Cobra `RunE`, minimal logic |
| `app` use-cases | Orchestrate detect/add flow and build execution plan | Plain Go services with constructor-injected dependencies |
| `domain/detectors` | Decide whether a provider matches + why | `Detector` interface returning structured evidence |
| `domain/selection` | Compute final provider set (`detected + include - exclude`) | Pure functions; deterministic sorting |
| `domain/managedblock` | Parse/replace managed region without touching user content | Marker-aware parser + renderer |
| `infra/gitignore` | Read/write `.gitignore` safely | Repository using `os` + atomic write strategy |
| `infra/toptal` | Fetch remote provider list + merged template content | HTTP client with explicit timeout + typed errors |
| `infra/probe` | OS/PATH/installed software probing | Small adapters behind interfaces |
| `output` | Human and `--json` response rendering | Presenter that maps use-case result to format |

## Recommended Project Structure

```text
cmd/
└── gitignore-gen/
    └── main.go                # bootstraps dependencies and executes root command

internal/
├── cli/
│   ├── root.go                # root command, persistent flags
│   ├── detect.go              # detect command wiring
│   └── add.go                 # add command wiring
├── app/
│   ├── detect_usecase.go      # detect orchestration
│   ├── add_usecase.go         # add orchestration
│   └── plan.go                # dry-run/write planning model
├── domain/
│   ├── detector/
│   │   ├── detector.go        # interface + result model
│   │   └── registry.go        # all registered detectors
│   ├── selection/
│   │   └── select.go          # detected/include/exclude + sorting
│   └── managedblock/
│       ├── parse.go           # marker discovery + extraction
│       └── render.go          # managed block rendering
├── infra/
│   ├── gitignore/
│   │   ├── repository.go      # read/write/migrate .gitignore
│   │   └── atomic.go          # safe write strategy
│   ├── toptal/
│   │   ├── client.go          # API integration
│   │   └── models.go          # remote payloads
│   └── probe/
│       ├── os_probe.go        # runtime OS detection
│       ├── path_probe.go      # PATH binary checks
│       └── app_probe.go       # known app install checks
└── output/
    ├── human.go               # concise terminal output
    └── json.go                # machine-readable result

testdata/
├── gitignore/                 # existing/missing/marker fixtures
├── api/                       # captured list/template responses
└── expected/                  # golden expected outputs
```

### Structure Rationale

- **`cli/` is transport only:** keeps Cobra details out of business logic, making use-cases unit-testable without command execution.
- **`app/` owns orchestration:** detect/add behaviors differ; separate use-cases keep each command policy explicit.
- **`domain/` stays pure:** selection, ordering, and managed-block rules are deterministic and should be testable without IO.
- **`infra/` isolates volatility:** filesystem, HTTP, and host probing are the unstable edges; isolating them prevents logic leakage.
- **`output/` is a boundary:** `--json` and human output can evolve without rewriting command/use-case code.

## Architectural Patterns

### Pattern 1: Command as Adapter, Use-Case as Core

**What:** Cobra command methods map inputs to a use-case request and pass through context.
**When to use:** Always for `detect` and `add`; avoid embedding business decisions in `RunE`.
**Trade-offs:** Slightly more boilerplate, much better testability and boundary clarity.

**Example:**
```go
func newDetectCmd(uc *app.DetectUseCase) *cobra.Command {
	cmd := &cobra.Command{
		Use: "detect",
		RunE: func(cmd *cobra.Command, _ []string) error {
			req := app.DetectRequest{/* map flags */}
			res, err := uc.Run(cmd.Context(), req)
			if err != nil { return err }
			return output.Render(cmd, res)
		},
	}
	return cmd
}
```

### Pattern 2: Plan-then-Apply (supports `--dry-run` naturally)

**What:** First compute a deterministic `Plan` (final providers, block text, file action), then optionally apply writes.
**When to use:** All mutating commands (`detect`, `add`).
**Trade-offs:** One extra model type; major simplification for dry-run and test assertions.

**Example:**
```go
type Plan struct {
	FinalProviders []string
	FileAction     string // create|insert|replace|noop
	NewContent     string
}

plan := planner.Build(currentFile, selection, template)
if req.DryRun { return result.FromPlan(plan), nil }
err := repo.Apply(plan)
```

### Pattern 3: Evidence-first Detection Results

**What:** Detectors return structured evidence (source/path/reason/error), not just bool.
**When to use:** Provider detection pipeline and verbose/JSON output.
**Trade-offs:** Slightly larger payloads; dramatically better explainability and diagnostics.

## Data Flow

### Detect Flow

```text
User: gitignore-gen detect [flags]
  ↓
CLI layer parses flags
  ↓
DetectUseCase.Run(ctx, request)
  ↓
DetectorRegistry executes project/env/global detectors
  ↓
SelectionEngine computes detected + include - exclude
  ↓
Sort provider keys alphabetically
  ↓
TemplateClient validates remote list + fetches merged template
  ↓
GitignoreRepository reads current file
  ↓
ManagedBlock parser builds replacement plan (create/insert/replace)
  ↓
if dry-run: return plan preview
else: apply write and return execution result
  ↓
Presenter renders human or JSON output
```

### Add Flow

```text
User: gitignore-gen add <keys...>
  ↓
CLI validates args
  ↓
AddUseCase loads existing managed provider set
  ↓
append only missing valid keys
  ↓
sort providers
  ↓
TemplateClient fetches merged template
  ↓
ManagedBlock replacement plan
  ↓
dry-run or apply
  ↓
output renderer
```

### State Management

The tool is stateless across runs. State exists only as:
1. Current filesystem state (`.gitignore`, project files)
2. Runtime environment probes (OS/PATH/apps)
3. Remote API responses in-memory for current execution

No persistent cache/config in v1 aligns with PRD scope.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Toptal gitignore API | Thin HTTP client behind `TemplateClient` interface | Hard-fail on API errors in v1; no fallback |

### Internal Boundaries (what talks to what)

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `cli` → `app` | Direct method calls with request structs | One-way only; app must not import Cobra |
| `app` → `domain` | Pure function calls | Deterministic core logic |
| `app` → `infra` | Interface-based adapters | Enables fake filesystem/API/probes in tests |
| `app` → `output` | Result DTOs | Output formatting decoupled from business logic |

## Suggested Build Order (dependency-aware)

1. **Domain primitives + deterministic selection rules**
   - Build `ProviderKey`, detection result/evidence, sorting, include/exclude merge logic.
   - Why first: everything else depends on a correct, testable provider set contract.

2. **Managed block parser/renderer + plan model**
   - Implement marker parsing and create/insert/replace planning as pure logic.
   - Why second: safety/idempotence is core product risk.

3. **Filesystem repository adapter**
   - Add read/apply implementation around the plan model.
   - Why now: enables end-to-end local behavior before network dependency.

4. **Template API client adapter**
   - Implement provider list check + template fetch + typed failures.
   - Why now: required for realistic command outcomes.

5. **Detector framework + initial detectors**
   - Registry + project/env/global detector sets returning evidence.
   - Why now: unlocks `detect` command semantics.

6. **Use-cases (`detect`, `add`)**
   - Orchestrate domain + infra; support dry-run and JSON metadata.
   - Why now: orchestration stabilizes once core and adapters exist.

7. **CLI wiring (Cobra commands + flags)**
   - Thin command adapters calling use-cases.
   - Why late: minimizes command churn while internals settle.

8. **Output presenters + fixture/golden tests**
   - Finalize human/JSON output and lock behavior with fixtures.
   - Why last: output shape depends on completed execution result model.

Dependency chain:

```text
Domain selection → Managed block planning → FS adapter
               ↘ API adapter + Detector framework ↘
                        Use-cases (detect/add) → CLI + output
```

## Anti-Patterns

### Anti-Pattern 1: Business logic inside Cobra `RunE`

**What people do:** Parse flags and also perform detection, API calls, and file writes directly in command handlers.
**Why it’s wrong:** Hard to test, impossible to reuse, and command files become brittle.
**Do this instead:** Keep `RunE` as argument adapter + use-case call only.

### Anti-Pattern 2: Mixing parse-and-write in one mutable function

**What people do:** Read file and immediately mutate bytes in-place while parsing markers.
**Why it’s wrong:** Easy to corrupt user content and hard to support dry-run.
**Do this instead:** Build immutable plan first, then apply in one write step.

### Anti-Pattern 3: Non-deterministic provider ordering

**What people do:** Preserve detector discovery order or map iteration order.
**Why it’s wrong:** Causes output churn and flaky tests.
**Do this instead:** Sort provider keys before storage, API request, and output.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Single repo / local use | Current architecture is sufficient (single-process orchestration) |
| Many runs in CI | Add request-level timeouts/retries and stricter output contracts; keep same layering |
| Very large monorepos (future) | Introduce scoped traversal engine + optional config, but keep detector/use-case boundaries |

## Sources

- https://cobra.dev/docs/how-to-guides/working-with-commands (HIGH: official Cobra docs; command organization and `RunE` patterns)
- https://cobra.dev/docs/how-to-guides/working-with-flags (HIGH: official Cobra docs; flag scoping/validation patterns)
- https://cobra.dev/docs/how-to-guides/context-and-tracing/ (MEDIUM: official docs; context propagation patterns)
- https://pkg.go.dev/github.com/spf13/cobra (HIGH: official API reference; `ExecuteContext`, `SetIn/SetOut`, command methods)
- https://go.dev/pkg/io/fs/ (HIGH: Go stdlib; filesystem abstraction and deterministic traversal semantics)
- https://pkg.go.dev/testing/fstest (HIGH: Go stdlib; in-memory FS and FS-contract testing helpers)
- https://github.com/cli/cli/blob/trunk/docs/project-layout.md (MEDIUM: production CLI reference architecture, command/use-case split and test guidance)
- https://www.toptal.com/developers/gitignore (MEDIUM: official service page and API usage entrypoint)
- https://www.toptal.com/developers/gitignore/api/go,node (MEDIUM: confirms merged-template API behavior)

---
*Architecture research for: gitignore-gen*
*Researched: 2026-04-07*
