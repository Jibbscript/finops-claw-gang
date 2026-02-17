from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable

from .models import ActionRiskLevel, ApprovalStatus, RISK_SCORE, RecommendedAction


@dataclass(frozen=True)
class PolicyDecision:
    approval: ApprovalStatus
    details: str


class PolicyEngine:
    """deterministic approval + safety policy. llms can *propose*, policy decides."""

    def __init__(
        self,
        auto_approve_max_risk: ActionRiskLevel = ActionRiskLevel.low,
        deny_min_risk: ActionRiskLevel = ActionRiskLevel.critical,
    ):
        self.auto_approve_max_risk = auto_approve_max_risk
        self.deny_min_risk = deny_min_risk

    def max_risk(self, actions: Iterable[RecommendedAction]) -> ActionRiskLevel:
        max_action = max(actions, key=lambda a: RISK_SCORE[a.risk_level])
        return max_action.risk_level

    def decide(self, actions: list[RecommendedAction]) -> PolicyDecision:
        if not actions:
            return PolicyDecision(ApprovalStatus.denied, "no recommended actions")

        max_risk = self.max_risk(actions)

        # hard deny critical
        if RISK_SCORE[max_risk] >= RISK_SCORE[self.deny_min_risk]:
            return PolicyDecision(
                ApprovalStatus.denied,
                f"critical-risk action(s) present: {max_risk.value}; manual-only",
            )

        # auto approve low risk
        if RISK_SCORE[max_risk] <= RISK_SCORE[self.auto_approve_max_risk]:
            return PolicyDecision(
                ApprovalStatus.auto_approved,
                f"auto-approved; max risk={max_risk.value}",
            )

        # otherwise require human
        return PolicyDecision(
            ApprovalStatus.pending,
            f"requires human approval; max risk={max_risk.value}",
        )


def enforce_executor_safety(
    approval: ApprovalStatus,
    actions: list[RecommendedAction],
    resource_tags_by_arn: dict[str, dict[str, str]],
) -> None:
    """raise ValueError if unsafe."""

    if approval not in (ApprovalStatus.approved, ApprovalStatus.auto_approved):
        raise ValueError(f"cannot execute: approval status is {approval.value}")

    for a in actions:
        if a.risk_level == ActionRiskLevel.critical:
            raise ValueError(f"refuse to execute critical action {a.action_id}")

        if a.target_resource:
            tags = resource_tags_by_arn.get(a.target_resource, {})
            if tags.get("do-not-modify") == "true" or tags.get("manual-only") == "true":
                raise ValueError(
                    f"refuse to execute on tagged resource {a.target_resource}: {tags}"
                )
