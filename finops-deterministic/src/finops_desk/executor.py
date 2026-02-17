from __future__ import annotations

from datetime import datetime, timezone

from .models import ApprovalStatus, ExecutionResult, RecommendedAction
from .policy import enforce_executor_safety
from .tools import InfraTools, now_utc_iso


class DeterministicExecutor:
    """no llm. just tools + explicit snapshots."""

    def __init__(self, infra: InfraTools):
        self.infra = infra

    def snapshot(self, action: RecommendedAction) -> dict:
        # in prod: describe_resource, asg config, etc.
        if action.target_resource:
            return {"tags": self.infra.resource_tags(action.target_resource)}
        return {}

    def execute_actions(
        self,
        approval: ApprovalStatus,
        actions: list[RecommendedAction],
        resource_tags_by_arn: dict[str, dict[str, str]],
    ) -> list[ExecutionResult]:
        enforce_executor_safety(approval, actions, resource_tags_by_arn)

        results: list[ExecutionResult] = []

        for a in actions:
            pre = self.snapshot(a)
            # stub: we do not actually call aws
            ok = True
            details = f"stub executed {a.action_type} on {a.target_resource}"
            post = pre

            results.append(
                ExecutionResult(
                    action_id=a.action_id,
                    executed_at=now_utc_iso(),
                    success=ok,
                    details=details,
                    rollback_available=True,
                    pre_action_snapshot=pre,
                    post_action_snapshot=post,
                )
            )

            if not ok:
                break

        return results
