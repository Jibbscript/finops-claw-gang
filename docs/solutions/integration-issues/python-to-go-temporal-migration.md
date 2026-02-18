---
title: "Python-to-Go Migration: LangGraph to Temporal (6 Phases)"
problem_type: integration_issue
component: finops-go
severity: n/a
tags:
  - migration
  - go
  - temporal
  - langgraph
  - python
  - architecture
date_solved: 2026-02-17
phases: 6
total_go_files: 102
total_go_tests: 39
total_packages: 36
binaries: 5
---

# Python-to-Go Migration: LangGraph to Temporal

Complete documentation of findings, insights, and reusable patterns from migrating finops-claw-gang from Python/LangGraph to Go/Temporal across 6 phases in a single day.

## Migration Summary

| Metric | Python (Before) | Go (After) |
|--------|-----------------|------------|
| Language | Python 3.10 | Go 1.24 |
| Orchestrator | LangGraph + MemorySaver | Temporal SDK v1.40.0 |
| Models | Pydantic v2 | Typed structs + JSON tags |
| State graph | `StateGraph` with 6 nodes | Sequential workflow function |
| Abstract classes | 3 fat interfaces | 10+ consumer-site interfaces |
| Checkpointing | In-memory (MemorySaver) | Durable event sourcing |
| HIL mechanism | `interrupt()` (blocks indefinitely) | Update handler + 24h timer |
| Source files | 13 Python files | 102 Go files across 36 packages |
| Test files | 3 test files | 39 test files |
| Binaries | 1 CLI | 5 (worker, cli, api, mcp, shadow-compare) |
| Frontend | None | Next.js schema-driven UI |
| Auth | None | OIDC/JWT |
| Observability | None | OpenTelemetry traces + metrics |

## Phase-by-Phase Timeline

### Phase 0: Project Setup & Python Baseline
**Commit**: `d54fcbd` | **Files**: 33 | **Lines**: +2,001

Established the Python LangGraph implementation as the migration source. Key artifacts:
- Complete Python anomaly desk in `finops-deterministic/`
- 10 JSON fixture files for all tool interfaces
- Migration spec documents (PDF, DOCX, RTF) pre-written
- CLAUDE.md with architectural principles ("no LLM in critical path")
- Go `.gitignore` created 14 minutes after Python init (migration was pre-planned)

### Phase 1+2: Deterministic Logic + Temporal Backbone
**Commit**: `d140d0e` | **Co-author**: Claude Opus 4.6 | **Files**: +47 | **Lines**: +4,564 | **Tests**: 57

Delivered as a single commit despite being planned as separate phases (11 + 8 AWUs). This was the right call -- the domain types and workflow logic are tightly coupled.

Phase 1 ported all deterministic logic:
- `internal/domain/` -- enums, models, validation (3 files)
- `internal/policy/` -- PolicyEngine with identical thresholds
- `internal/triage/` -- priority-ordered classifier
- `internal/analysis/` -- deterministic planner
- `internal/executor/` -- safety-gated executor
- `internal/verifier/` -- health checker

Phase 2 added the Temporal backbone:
- `internal/temporal/workflows/` -- AnomalyLifecycleWorkflow replacing StateGraph
- `internal/temporal/activities/` -- activity bridge with union interfaces
- `cmd/worker-finops/` and `cmd/cli/` -- first two binaries
- `internal/testutil/stubs.go` -- golden fixture loader
- HIL via Update handler with 24h timer + Selector

### Phase 3: Real AWS Connectors
**PR**: #2 | **Co-author**: Claude Opus 4.6 | **Files**: +23 | **Lines**: +1,743 | **Tests**: 80

Zero interface changes to Phase 1+2 code. Connectors satisfied existing interfaces:
- 5 AWS service packages: `costexplorer`, `athena`, `cloudwatch`, `tagging`, `codedeploy`
- KubeCost HTTP client
- Composite adapters: `AWSCostClient`, `AWSInfraClient`
- Config-driven factory: `FINOPS_MODE=stub|production`

Key insight: The consumer-site interface pattern paid off here. New connector packages implemented existing narrow interfaces with no changes to consuming code.

### Phase 4: AWS-Doctor Integration
**PR**: #3 | **Co-author**: Claude Sonnet 4.5 | **Files**: +20 | **Lines**: +1,318 | **Tests**: 99 (316 subtests)

Integrated CLI-only tool via `os/exec` wrapper:
- `internal/connectors/awsdoctor/` -- BinaryRunner, typed output parsing
- Waste check at triage priority 3 (shifted existing priorities 3-8 to 4-9)
- Template-based waste analysis in `analysis.AnalyzeWaste()`
- `AWSDocSweepWorkflow` for multi-account scheduled scans
- Nil-safe `WasteQuerier` interface for backward compatibility

Review feedback commit addressed 7 specific findings, growing tests from 91 to 99.

### Phase 5: Generative UI
**PR**: #4 | **Co-author**: Claude Sonnet 4.5 | **Files**: +43 | **Lines**: +3,084 | **Tests**: 114

Largest phase by file count. Introduced 4 new Go packages + 2 binaries + Next.js frontend:
- `internal/api/` -- HTTP API with 7 routes (stdlib `net/http`)
- `internal/agui/` -- AG-UI SSE streaming (poll-based state change detection)
- `internal/uischema/` -- 14 component types, schema-driven UI contract
- `internal/mcpserver/` -- 5 MCP tools via stdio transport
- `internal/temporal/querier/` -- WorkflowQuerier interface
- `web/` -- Next.js frontend with `componentRegistry` + `ComponentRenderer`

Key insight: Query handler must be registered before the first blocking call in the workflow, otherwise queries fail for running workflows.

### Phase 6: Production Hardening
**PR**: #5 | **Co-author**: Claude Opus 4.6 | **Files**: +40 | **Lines**: +2,520 | **Tests**: 27 packages

Final phase delivered 9 AWUs covering all production concerns:
- `internal/temporal/queues/` -- 3 task queues with permission isolation
- `internal/ratelimit/` -- token bucket per service + per-tenant activity budgets
- `internal/api/auth.go` -- OIDC/JWT middleware
- `internal/connectors/aws/tenant_auth.go` -- STS session cache
- `internal/shadow/` -- Go-vs-Python comparison tool (CI gate)
- `internal/observability/` -- `log/slog` + OpenTelemetry
- Dockerfile (multi-stage, 5 binaries) + docker-compose.yml
- Cutover runbook + deployment docs

Review feedback addressed 8 findings: context safety, OIDC error handling, table-driven tests, `t.Parallel()`.

---

## Translation Patterns

### Pydantic Models to Go Structs

```python
# Python
class CostAnomaly(BaseModel):
    anomaly_id: str = Field(default_factory=lambda: str(uuid.uuid4())[:8])
    service: str = ""
    delta_dollars: float = 0.0
```

```go
// Go
type CostAnomaly struct {
    AnomalyID    string  `json:"anomaly_id"`
    Service      string  `json:"service"`
    DeltaDollars float64 `json:"delta_dollars"`
}
```

Rules:
- `Field(default_factory=...)` becomes a `New*()` constructor; Go zero values cover most cases
- `model_dump()` replaced by `json.Marshal` via struct tags
- JSON tags guarantee wire-format compatibility with Python fixtures

### Python Enums to Go Typed Constants

```python
# Python
class AnomalySeverity(str, Enum):
    low = "low"
    medium = "medium"
```

```go
// Go
type AnomalySeverity string
const (
    SeverityLow    AnomalySeverity = "low"
    SeverityMedium AnomalySeverity = "medium"
)
func (s AnomalySeverity) Valid() bool { /* switch statement */ }
```

Rules:
- `str, Enum` becomes `type X string` with named constants
- Each enum type gets a `Valid() bool` method (Python gets this free from `Enum`)
- String values kept byte-identical for JSON/fixture compatibility
- Contract tests assert Go values match Python exactly

### Optional[float] to Go Pointers

```python
ri_coverage_delta: Optional[float] = None
```

```go
RICoverageDelta *float64 `json:"ri_coverage_delta"`
```

Semantic distinction: `nil` = "not measured" vs. `0.0` = "measured as zero". Helper: `func float64Ptr(v float64) *float64 { return &v }`.

### LangGraph StateGraph to Temporal Workflow

The most architecturally significant translation:

| LangGraph | Temporal |
|-----------|----------|
| `StateGraph` with nodes + conditional edges | Sequential function with `if/switch` + early returns |
| Mutable shared `FinOpsState` dict | Local `state` variable returned in `WorkflowResult` |
| `MemorySaver` checkpointing | Built-in event sourcing |
| 6 routing functions | Inline `if` checks |
| Node factories (closures over `Runtime`) | Activities with typed input structs |
| `interrupt()` (blocks indefinitely) | Update handler + Validator + Selector + Timer |

The linear Go workflow function is more readable than the implicit graph topology for this use case.

### ABC/Protocol to Consumer-Site Interfaces

Python's 3 fat abstract classes split into 6+ narrow Go interfaces at consumer sites:

| Python | Go |
|--------|-----|
| `CostTools` (6 methods) | `triage.CostFetcher` (3), `analysis.CostQuerier` (1), `verifier.CostChecker` (1) |
| `InfraTools` (3 methods) | `executor.TagFetcher` (1) |
| `KubeCostTools` (1 method) | `triage.KubeCostFetcher` (1) |

Union interfaces at activity boundary (`activities.CostDeps`) compose them back for DI.

Note: 2 of 6 Python `CostTools` methods (`get_ri_utilization`, `get_sp_utilization`) have no Go consumer interface -- they exist in connectors but no package declares a need for them. This is the consumer-site pattern working as intended: unused methods are not forced into interfaces.

### Domain Model Extensions (Not Just Translation)

Phase 4 added Go-only domain concepts that have no Python equivalent:
- `CategoryResourceWaste` enum value -- new triage category for waste detection
- `WasteFinding` struct and `WasteFindings []WasteFinding` on `TriageEvidence`
- `WasteSavings *float64`, `TrendVelocityPct *float64`, `TrendDirection string` fields

This shows the migration was not purely a port -- it also extended the domain model where Go capabilities enabled new features.

### Validation as a Separate Concern

Python gets validation "for free" from Pydantic at construction time. Go requires explicit validation functions in `domain/validate.go`: `ValidateCostAnomaly`, `ValidateRecommendedAction`, `ValidateTriageResult`, `ValidateTenantContext`, `ValidateVerificationResult`, `ValidateFinOpsState`. These can be called at integration boundaries (API handlers, activity inputs) without reconstructing objects.

### Error Handling

Python exceptions become Go `(T, error)` return tuples:
- `ValueError` in `enforce_executor_safety()` becomes `error` return
- Consistent `fmt.Errorf("context: %w", err)` wrapping at every call site
- Business failures return `WorkflowResult` with error reason; only infra failures produce workflow-level errors

---

## Key Design Decisions

### 1. Consumer-Site Interfaces (Not One Fat Interface)

Each Go package declares only the dependency surface it actually uses. Benefits:
- Mocks for `verifier` tests only implement `GetCostTimeseries`, not 6 methods
- Package dependency graphs are visible from interface files
- Breaking changes are scoped per consumer

Trade-off: Union interfaces (`activities.CostDeps`) needed at the activity boundary. Resolved by Go's implicit interface satisfaction.

### 2. RISK_SCORE as Map (Not Enum Ordering)

```go
var RiskScore = map[ActionRiskLevel]int{
    RiskLow: 10, RiskLowMedium: 20, RiskMedium: 30, RiskHigh: 40, RiskCritical: 50,
}
```

Adding a new level between existing ones cannot silently change comparisons. The policy engine uses `domain.RiskScore[risk]` for all comparisons, never `risk > otherRisk`. Contract tests assert exact numeric values matching Python.

### 3. HIL via Update (Not Signal)

Updates are synchronous (caller gets confirmation); Signals are fire-and-forget. The implementation adds three capabilities Python lacked:
1. **Validation**: Rejects malformed requests before processing
2. **Timeout**: 24h timer races against human response
3. **Idempotency**: `responded` flag prevents double-processing

### 4. Policy In-Workflow (Determinism-Safe)

`PolicyEngine.Decide()` is pure (no I/O, no side effects). Runs directly in the workflow, not as an activity. Avoids unnecessary serialization overhead and extra events in workflow history.

### 5. MaximumAttempts: 1 (No Retries for Execution)

For a system that modifies cloud infrastructure, automatic retries are dangerous. A failed `ExecuteActions` activity might have partially modified resources. The system captures errors in state and lets humans decide.

### 6. WorkflowResult on All Paths

```go
func AnomalyLifecycleWorkflow(...) (WorkflowResult, error) {
    return WorkflowResult{State: state, Reason: ReasonNoAnomaly}, nil        // business outcome
    return WorkflowResult{}, fmt.Errorf("...")                               // infra failure only
}
```

11 explicit `TerminationReason` values document every exit path. Business failures are successful completions with details in state.

### 7. Workflow Versioning for Safe Rollouts

`workflow.GetVersion(ctx, "exec-queue-routing", workflow.DefaultVersion, 1)` safely routes execution activities to the `QueueExec` queue only for new workflow executions. In-flight workflows continue using the old queue. This Temporal-specific pattern prevents replay failures when changing activity routing.

Version strings and queue names are centralized in `internal/temporal/versioning/versioning.go` as a single source of truth.

### 8. Schema-Driven UI

Backend emits `UISchema`; frontend renders dynamically via `componentRegistry`. Decouples backend logic from frontend rendering. Same schema can drive web, mobile, or MCP tool output.

### 8. Task Queue Partitioning

Three queues with permission isolation:
- `finops-anomaly` (10 concurrent) -- workflows
- `finops-detect` (20 concurrent) -- read-heavy detection on read-only workers
- `finops-exec` (3 concurrent) -- restricted writes on write-credential workers

---

## Testing Strategy

### Golden Fixtures

`tests/golden/` contains byte-identical copies of Python `tools/fixtures/`. Found via `testutil.GoldenDir()` using `runtime.Caller(0)` to locate fixtures relative to the source file -- no environment variables or build flags needed. This pattern survives refactoring and works without any build configuration, unlike the common alternatives of env vars or `go:embed`.

### Contract Tests (Enum Parity)

`domain/enums_test.go` asserts Go constant string values match Python exactly:
- `TestAnomalySeverityStringValues`: `SeverityLow == "low"`
- `TestRiskScoreMap`: `RiskScore[RiskLow] == 10`

### Temporal Workflow Tests

`WorkflowTestSuite` mocks activities by name, tests 10 scenarios:
1. Happy path (auto-approved)
2. Expected growth early exit
3. No actions early exit
4. Policy denied (critical risk)
5. HIL approved / denied / timed out
6. Activity errors (triage, execution)
7. No anomaly

HIL tests use `RegisterDelayedCallback` + `UpdateWorkflowNoRejection`.

### Table-Driven Tests with t.Parallel()

All test files use the table-driven pattern with `t.Run()` and `t.Parallel()` for concurrent execution and clear isolation.

### Mock Patterns

| Pattern | Use Case | Example |
|---------|----------|---------|
| Stub implementations | Golden fixture loading | `StubCost` satisfies `CostFetcher`, `CostQuerier`, `CostChecker` |
| Shell script mocks | CLI tool wrapping | Temp scripts that `cat` fixture files for aws-doctor tests |
| Narrow API mocks | AWS SDK unit tests | Mock 3-method `costexplorer.API`, not the full SDK client |
| Activity name mocks | Workflow logic testing | `env.OnActivity("TriageAnomaly", ...)` |

### Shadow-Run Parity Testing

`cmd/shadow-compare/` runs both Go and Python pipelines on identical fixtures, compares JSON output phase-by-phase. Exits non-zero on divergence (CI gate).

---

## Architecture Improvements Over Python

### What Got Better

1. **I/O separation from logic**: Temporal enforces the boundary (activities for I/O, pure functions in-workflow)
2. **Durability**: Event-sourced replay vs. in-memory checkpointing
3. **Typed termination**: 11 explicit `TerminationReason` values vs. Python's undifferentiated "end"
4. **Validation as separate concern**: Explicit `validate.go` functions callable at integration boundaries
5. **Multi-tenancy**: `TenantClientFactory` with STS AssumeRole (Python had none)
6. **Rate limiting**: Per-service token bucket + per-tenant activity budgets (Python had none)
7. **Task queue partitioning**: Permission isolation between read and write workers

### What Got More Complex

1. **Interface system**: 3 abstract classes became 10+ interfaces + compile-time checks (more boilerplate, better safety)
2. **HIL implementation**: `interrupt()` (1 line) became `waitForApproval` (50 lines with Update + Validator + Selector + Timer)
3. **Connector layer**: Stubs-only became 5 AWS packages + composite adapters + tenant factory + rate limiters

### Patterns That Emerged (Not in Python)

- Composite adapters composing multiple AWS clients into activity interfaces
- Union interfaces at activity boundaries
- Schema-driven UI rendering contract (14 component types)
- AG-UI SSE streaming protocol
- Shadow-run comparison tooling (migration-specific)
- OIDC authentication layer
- OpenTelemetry observability

---

## Prevention Strategies & Best Practices

### For Future Language Migrations

1. **Pre-write the migration spec before touching code.** Migration specs were committed at project inception. This eliminated scope creep and provided clear phase boundaries.

2. **Combine tightly-coupled phases.** Phases 1+2 were planned separately but delivered together. Domain types and workflows that use them are too coupled to deliver independently.

3. **Maintain byte-identical fixtures.** Golden fixtures shared between Python and Go enabled shadow-run parity testing. Never transform fixtures during migration.

4. **Use contract tests for wire format parity.** Explicit tests asserting `Go_constant == "python_string_value"` catch silent breakage from enum or constant changes.

5. **Design interfaces at the consumer, not the provider.** Consumer-site interfaces made connectors a zero-change addition in Phase 3. If interfaces had been defined at the provider, Phase 3 would have required refactoring Phase 1 code.

6. **Review gates between phases catch real issues.** Phases 4 and 6 had explicit review-driven fix commits addressing 7 and 8 findings respectively: error handling, test coverage, code deduplication, Go idioms.

7. **Shadow-run before cutover.** The `shadow-compare` tool validates Go-vs-Python parity offline. Build the comparison tool early enough to use it as a CI gate before production cutover.

### Reusable Patterns

**Wrapping CLI tools safely (os/exec):**
- Define a `Runner` interface with typed output
- `BinaryRunner` uses `exec.CommandContext` for cancellation
- Parse stdout as JSON into typed structs at the connector boundary
- Test with shell script mocks that `cat` fixture files
- Handle: missing binary, non-zero exit, invalid JSON

**Bridging interrupt() to Temporal HIL:**
- Use Update (not Signal) for synchronous confirmation
- Add a Validator for input validation before processing
- Race a Timer against the response via Selector
- Guard against duplicate responses with a `responded` flag

**Adding tenant isolation incrementally:**
- Start with `TenantContext` as data (Phase 1)
- Add `TenantClientFactory` with STS AssumeRole (Phase 6)
- Wire via `activities.TenantDeps` interface (Phase 6)
- No changes needed to workflow or domain logic

**Adding rate limiting without changing interfaces:**
- Embed rate limiter in composite adapters (call `limiter.Wait()` before each method)
- `context.WithTimeout` safety net on `limiter.Wait()` calls
- Per-tenant activity budgets as a separate concern from per-service rate limits

### Anti-Patterns Avoided

| Anti-Pattern | What Was Done Instead |
|---|---|
| Fat interface (one big `CostTools` with 6 methods) | Consumer-site interfaces (1 narrow interface per consumer package, 6+ total) |
| Enum ordering for risk comparison | Explicit `RiskScore` map with contract tests |
| Retrying destructive operations | `MaximumAttempts: 1` -- capture error, let humans decide |
| Silent error degradation | `api.New()` returns error on OIDC failure; `slog.Warn` on invalid config |
| Tight UI-backend coupling | Schema-driven UI; backend emits schema, frontend renders |
| Mocking entire SDK clients | Narrow `API` interfaces (3-5 methods per service) |
| `--no-verify` or hook bypasses | All review findings addressed with new commits |

---

## Codebase Growth Trajectory

| Phase | Files | Go Files | Tests | Test Count | Net Lines |
|-------|-------|----------|-------|------------|-----------|
| 0 | 33 | 0 | 0 | 3 (Python) | +2,001 |
| 1+2 | 82 | 35 | 15 | 57 | +4,564 |
| 3 | 105 | 55 | 24 | 80 | +1,743 |
| 4 | 112 | 60 | 26 | 99 (316 subtests) | +1,537 |
| 5 | 151 | 80 | 31 | 114 | +3,081 |
| 6 | 177 | 102 | 39 | 27 packages | +2,441 |

**Packages**: 12 (Phase 1+2) -> 20 (Phase 3) -> 23 (Phase 4) -> 31 (Phase 5) -> 36 (Phase 6)

**Binaries**: 2 (worker, cli) -> 4 (+api, mcp) -> 5 (+shadow-compare)

**Hot files** (modified in 4+ phases):
- `cmd/worker-finops/main.go` -- 5 phases (integration point for all new features)
- `internal/temporal/activities/activities.go` -- 5 phases
- `internal/config/config.go` -- 4 phases
- `go.mod` -- 4 phases

## AI Co-Author Patterns

- **Claude Opus 4.6**: Phases 0, 1+2, 3, 6 (foundational + production hardening)
- **Claude Sonnet 4.5**: Phases 4, 5 (integration + UI layer)
- Both had review-driven follow-up commits incorporated

---

## Cross-References

- Migration spec: `docs/plans/2026-02-17-refactor-python-to-go-migration-plan.md`
- Deployment guide: `finops-go/docs/deployment.md`
- Cutover runbook: `finops-go/docs/cutover-runbook.md`
- Python design doc: `finops-deterministic/docs/deterministic-redesign.md`
- Project instructions: `CLAUDE.md`
- PRs: #2 (Phase 3), #3 (Phase 4), #4 (Phase 5), #5 (Phase 6)
