from finops_desk.models import RecommendedAction, ActionRiskLevel, ApprovalStatus
from finops_desk.policy import PolicyEngine


def test_policy_auto_approves_low():
    pe = PolicyEngine(auto_approve_max_risk=ActionRiskLevel.low)
    actions = [
        RecommendedAction(
            description="tag resource",
            action_type="tag",
            risk_level=ActionRiskLevel.low,
            rollback_procedure="remove tag",
        )
    ]
    d = pe.decide(actions)
    assert d.approval == ApprovalStatus.auto_approved


def test_policy_denies_critical():
    pe = PolicyEngine()
    actions = [
        RecommendedAction(
            description="terminate prod",
            action_type="terminate",
            risk_level=ActionRiskLevel.critical,
            rollback_procedure="n/a",
        )
    ]
    d = pe.decide(actions)
    assert d.approval == ApprovalStatus.denied
