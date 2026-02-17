# deterministic-first redesign (spec)

## what we ripped out (dangerous)

1. executor-as-llm: removed. executor is now deterministic tool calls + snapshots.
2. enum-order risk max: removed. uses explicit numeric `RISK_SCORE`.
3. fake hil: removed. uses langgraph `interrupt()` to *actually stop* execution.
4. prose outputs: removed. nodes produce typed pydantic objects.

## control plane rules

- llms may propose narratives and candidate actions, but:
  - approval is decided by policy-as-code
  - execution is performed by deterministic code
  - safety gates are enforced in code (tags, critical denial, snapshot requirement)

## hil flow

- node decides `approval=pending` -> `interrupt(payload)`
- resumption supplies `{approve: bool, by: str}`
- state continues from checkpoint

## triage primitives (new)

- commitment coverage drift:
  - reserved instances: coverage + utilization
  - savings plans: coverage + utilization
- credits/refunds/fees:
  - from cur line_item_type (credit/refund/fee/rifee)
- marketplace:
  - from cur product metadata / product code
- data transfer:
  - usage types with `DataTransfer-*` (e.g., `REGION-DataTransfer-Out-Bytes`)
- kubecost:
  - allocation deltas by namespace/controller/label

