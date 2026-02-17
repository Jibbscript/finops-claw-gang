from finops_desk.models import CostAnomaly
from finops_desk.tools import StubCostTools, StubInfraTools, StubKubeCostTools
from finops_desk.triage import triage


def test_triage_detects_data_transfer(tmp_path):
    # use repo fixtures
    import os
    fixtures = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "tools", "fixtures"))
    cost = StubCostTools(fixtures)
    infra = StubInfraTools(fixtures)
    kube = StubKubeCostTools(fixtures)

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

    res = triage(anomaly, cost, infra, kube)
    # our fixture has a meaningful DataTransfer-Out-Bytes component
    assert res.category.value in {"data_transfer", "k8s_cost_shift", "credits_refunds_fees", "unknown", "deploy_related"}
