# finops-desk (deterministic-first)

this is a runnable skeleton that refactors the original langgraph finops workflow spec into a deterministic-first control plane:

- llm-free triage + approvals + execution (llm optional later for narrative only)
- explicit finops primitives: ri/sp coverage + utilization, credits/refunds/fees, marketplace, data transfer, kubecost allocation
- real hil uses langgraph `interrupt()` so the graph **stops** until resumed

## quickstart

```bash
python -m venv .venv && source .venv/bin/activate
pip install -e .

# run the skeleton with stub fixtures
finops-desk --fixtures tools/fixtures --service EC2 --delta 750

# run tests
pip install -r requirements-dev.txt
pytest -q
```

## where to plug in real systems

- replace `StubCostTools` with real aws cost explorer + cur/athena + savings plans / ri calls
  - cost explorer: GetCostAndUsage, group by dimensions like SERVICE/USAGE_TYPE/RECORD_TYPE
  - savings plans: GetSavingsPlansCoverage / GetSavingsPlansUtilization
  - reserved instances: GetReservationCoverage / GetReservationUtilization
- replace `StubKubeCostTools` with kubecost allocation api `/model/allocation`

