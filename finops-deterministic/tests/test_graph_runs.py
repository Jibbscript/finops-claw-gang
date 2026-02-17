import os

from finops_desk.graph import Runtime, build_app
from finops_desk.models import CostAnomaly, FinOpsState, TenantContext
from finops_desk.tools import StubCostTools, StubInfraTools, StubKubeCostTools


def test_graph_stream_completes_without_interrupt():
    fixtures = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "tools", "fixtures"))
    rt = Runtime(StubCostTools(fixtures), StubInfraTools(fixtures), StubKubeCostTools(fixtures))
    app = build_app(rt)

    anomaly = CostAnomaly(
        service="EC2",
        account_id="123456789012",
        region="us-east-1",
        team="platform",
        expected_daily_cost=2400.0,
        actual_daily_cost=3150.0,
        delta_dollars=750.0,
        delta_percent=31.25,
        z_score=3.2,
    )

    state = FinOpsState(tenant=TenantContext(tenant_id="t"), anomaly=anomaly)
    config = {"configurable": {"thread_id": state.workflow_id}}

    last = None
    for event in app.stream(state, config=config):
        for _, out in event.items():
            last = out

    assert last is not None
    assert last.current_phase in {"verifier", "executor", "hil_gate", "analyst", "triager", "watcher"}
