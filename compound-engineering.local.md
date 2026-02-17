---
review_agents: [kieran-python-reviewer, code-simplicity-reviewer, security-sentinel, performance-oracle, architecture-strategist]
plan_review_agents: [kieran-python-reviewer, code-simplicity-reviewer]
---

# Review Context

This is a deterministic-first FinOps anomaly desk built on LangGraph (Python). The codebase is migrating toward Go 1.25+.

- **Python code** (finops-deterministic/): LangGraph StateGraph, Pydantic v2 models, deterministic triage/policy/execution. No LLM in the critical path.
- **Go code** (future): Will follow Effective Go idioms. The global `go-reviewer` skill is installed at ~/.claude/skills/go-reviewer/ for Go-specific review.
- **Security-critical**: The executor runs actions against AWS resources. `enforce_executor_safety` in policy.py is a hard gate — never weaken it. Tag-based protection (`do-not-modify`, `manual-only`) must be preserved.
- **Policy-as-code**: Approval decisions are deterministic (PolicyEngine). LLMs propose only; code decides.
- **HIL via interrupt()**: Human-in-the-loop uses real LangGraph `interrupt()`, not stubs. The graph actually stops.
