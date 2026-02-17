---
title: "Python-to-Go Migration: FinOps Anomaly Desk"
type: refactor
status: active
date: 2026-02-17
---

# Python-to-Go Migration: FinOps Deterministic Anomaly Desk

See the full plan at `.claude/plans/purring-meandering-hedgehog.md` for the active working copy.

This plan covers the migration of the Python LangGraph-based FinOps anomaly desk to Go v1.25+ with Temporal workflows. It contains 50 atomic work units across 7 phases:

- **Phase 0**: Parity harness & contracts (5 AWUs)
- **Phase 1**: Domain & deterministic engines in Go (11 AWUs)
- **Phase 2**: Temporal backbone (8 AWUs)
- **Phase 3**: Real data connectors & MCP servers (9 AWUs)
- **Phase 4**: AWS Doctor integration (4 AWUs)
- **Phase 5**: Generative UI (6 AWUs)
- **Phase 6**: Production hardening & cutover (9 AWUs)

Critical path: 12 sequential AWUs. Extensive parallelization possible within each phase.
