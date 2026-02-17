# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**finops-claw-gang** is a deterministic-first FinOps anomaly desk built on LangGraph. The core principle: LLMs may propose narratives and candidate actions, but approval is decided by policy-as-code, execution is performed by deterministic code, and safety gates are enforced in code.

The runnable code lives in `finops-deterministic/`. There are also migration spec documents at the repo root (`migration-spec.pdf`, `migration-spec.docx`, `migration-spec-with-citations.rtf`).

## Build & Run

```bash
cd finops-deterministic
python -m venv .venv && source .venv/bin/activate
pip install -e .
pip install -r requirements-dev.txt

# run with stub fixtures
finops-desk --fixtures tools/fixtures --service EC2 --delta 750

# run all tests
pytest -q

# run a single test
pytest tests/test_policy.py::test_policy_auto_approves_low -v
```

Requires Python >=3.10. Key dependencies: langgraph, langchain-core, pydantic v2, PyYAML.

## Architecture

The system is a LangGraph `StateGraph` with `FinOpsState` (Pydantic model) as the shared state object. All graph nodes are defined in `graph.py` and wired with conditional edges.

### Graph Flow

```
watcher -> triager -> analyst -> hil_gate -> executor -> verifier -> END
```

Each node can short-circuit to END via routing functions (e.g., expected growth with high confidence exits after triage; no recommended actions exits after analyst; denied approval exits after hil_gate).

### Key Modules (all in `finops-deterministic/src/finops_desk/`)

- **models.py** — All Pydantic state objects and enums. `FinOpsState` is the LangGraph state schema. `RISK_SCORE` maps `ActionRiskLevel` enums to explicit numeric scores (not enum ordering).
- **triage.py** — Deterministic classifier with priority-ordered evidence checks: commitment coverage drift (RI/SP) > credits/refunds/fees > marketplace > data transfer > k8s allocation > deploy correlation > expected growth > config drift. No LLM involved.
- **policy.py** — `PolicyEngine` decides auto-approve / pending / deny based on `RISK_SCORE` thresholds. `enforce_executor_safety()` is a hard gate that raises `ValueError` on critical actions or tagged resources (`do-not-modify`, `manual-only`).
- **tools.py** — Abstract interfaces (`CostTools`, `InfraTools`, `KubeCostTools`) with `Stub*` implementations that read from `tools/fixtures/*.json`. Production: replace stubs with AWS Cost Explorer, CUR/Athena, Savings Plans, RI APIs, and KubeCost allocation API.
- **executor.py** — `DeterministicExecutor` takes pre/post snapshots and calls `enforce_executor_safety` before any action. Currently stubbed (no real AWS calls).
- **verifier.py** — Post-execution verification: checks service health and observed cost reduction, recommends close/rollback/escalate/monitor.
- **analysis.py** — Placeholder deterministic analyst. In production, LLM adds narrative but actions must still pass policy validation.
- **graph.py** — `Runtime` holds tool instances + policy engine. `build_app()` compiles the StateGraph with `MemorySaver` checkpointer. Node factories close over `Runtime`.
- **cli.py** — CLI entry point (`finops-desk` command). Streams graph events and prints final state.

### Human-in-the-Loop (HIL)

Uses LangGraph `interrupt()` in `hil_gate_node`. When policy returns `pending`, the graph stops and waits for external resumption with `{"approve": true/false, "by": "..."}`. The CLI auto-proceeds (no interactive approval in stub mode).

### Test Fixtures

`tools/fixtures/` contains JSON stubs for all tool interfaces: CUR line items, cost timeseries, RI/SP coverage and utilization, CloudWatch metrics, deploys, resource tags, and KubeCost allocation. Tests and CLI both use these.

## Design Decisions to Preserve

- **No LLM in the critical path**: Triage, policy, execution, and verification are all deterministic. LLM is optional for narrative only.
- **Explicit risk scores**: `RISK_SCORE` dict maps enums to ints. Never rely on enum ordering.
- **Real HIL via `interrupt()`**: The graph actually stops execution. No fake approval stubs in the graph itself.
- **Typed outputs**: Every node produces Pydantic objects, not prose strings.
- **Safety enforcement is code**: Tag-based protection (`do-not-modify`, `manual-only`), critical action denial, and approval status checks are all in `enforce_executor_safety`.
