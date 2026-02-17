from __future__ import annotations

import argparse
import json
import os

from .graph import Runtime, build_app
from .models import CostAnomaly, FinOpsState, TenantContext
from .tools import StubCostTools, StubInfraTools, StubKubeCostTools


def main() -> None:
    ap = argparse.ArgumentParser(prog="finops-desk")
    ap.add_argument("--fixtures", default=os.path.join(os.path.dirname(__file__), "..", "..", "tools", "fixtures"))
    ap.add_argument("--tenant", default="tenant-001")
    ap.add_argument("--account", default="123456789012")
    ap.add_argument("--service", default="EC2")
    ap.add_argument("--delta", type=float, default=750.0)
    args = ap.parse_args()

    fixtures = os.path.abspath(args.fixtures)

    cost = StubCostTools(fixtures)
    infra = StubInfraTools(fixtures)
    kube = StubKubeCostTools(fixtures)

    rt = Runtime(cost=cost, infra=infra, kubecost=kube)
    app = build_app(rt)

    anomaly = CostAnomaly(
        service=args.service,
        account_id=args.account,
        region="us-east-1",
        team="platform",
        expected_daily_cost=2400.0,
        actual_daily_cost=2400.0 + args.delta,
        delta_dollars=args.delta,
        delta_percent=(args.delta / 2400.0) * 100.0,
        z_score=3.2,
    )

    state = FinOpsState(tenant=TenantContext(tenant_id=args.tenant), anomaly=anomaly)

    config = {"configurable": {"thread_id": state.workflow_id}}

    # run; if it hits hil interrupt, auto-deny by default in this cli
    final = None
    for event in app.stream(state, config=config):
        for node, out in event.items():
            final = out
            print(f"[{node}] done")

    if final:
        print(json.dumps(final.model_dump(), indent=2))


if __name__ == "__main__":
    main()
