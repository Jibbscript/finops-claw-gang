from __future__ import annotations

from datetime import datetime, timezone

from .models import AnalysisResult, RecommendedAction, ActionRiskLevel
from .tools import CostTools, InfraTools


# placeholder: deterministic analyst that proposes safe actions only when evidence is strong.
# in prod, you'd use llm for narrative, but actions must be policy-validated + tool-verifiable.


def analyze_and_recommend(
    *,
    account_id: str,
    service: str,
    window_start: str,
    window_end: str,
    cost: CostTools,
    infra: InfraTools,
) -> AnalysisResult:
    cur = cost.get_cur_line_items(account_id, window_start, window_end, service=service)

    # extremely naive: if we see "idle" tag or obvious unused, propose tagging/budget alert.
    narrative = f"cur line items reviewed for {service} {window_start}..{window_end}; further attribution required"

    actions: list[RecommendedAction] = []

    # safe default: create budget alert (low risk)
    actions.append(
        RecommendedAction(
            description=f"create/update budget alert for {service} to catch recurrence",
            action_type="create_budget_alert",
            risk_level=ActionRiskLevel.low,
            estimated_savings_monthly=0.0,
            target_resource=f"budget:{service}:{account_id}",
            parameters={"amount": 0.0, "threshold_percent": 20.0},
            rollback_procedure="disable alert / delete budget rule",
        )
    )

    return AnalysisResult(
        root_cause_narrative=narrative,
        affected_resources=[],
        recommended_actions=actions,
        estimated_monthly_savings=0.0,
        confidence=0.4,
    )
