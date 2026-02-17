from __future__ import annotations

import uuid
from dataclasses import dataclass
from datetime import datetime, timezone
from enum import Enum
from typing import Any, Literal, Optional

from pydantic import BaseModel, Field


# --- enums ---
class AnomalySeverity(str, Enum):
    low = "low"
    medium = "medium"
    high = "high"
    critical = "critical"


class AnomalyCategory(str, Enum):
    expected_growth = "expected_growth"
    deploy_related = "deploy_related"
    config_drift = "config_drift"
    pricing_change = "pricing_change"
    credits_refunds_fees = "credits_refunds_fees"
    marketplace = "marketplace"
    data_transfer = "data_transfer"
    k8s_cost_shift = "k8s_cost_shift"
    commitment_coverage_drift = "commitment_coverage_drift"  # ri/sp
    unknown = "unknown"


class ApprovalStatus(str, Enum):
    pending = "pending"
    approved = "approved"
    denied = "denied"
    auto_approved = "auto_approved"
    timed_out = "timed_out"


class ActionRiskLevel(str, Enum):
    low = "low"
    low_medium = "low_medium"
    medium = "medium"
    high = "high"
    critical = "critical"


RISK_SCORE: dict[ActionRiskLevel, int] = {
    ActionRiskLevel.low: 10,
    ActionRiskLevel.low_medium: 20,
    ActionRiskLevel.medium: 30,
    ActionRiskLevel.high: 40,
    ActionRiskLevel.critical: 50,
}


# --- core state objects ---
class CostAnomaly(BaseModel):
    anomaly_id: str = Field(default_factory=lambda: str(uuid.uuid4())[:8])
    detected_at: str = Field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

    service: str = ""
    account_id: str = ""
    region: str = ""
    team: str = ""

    expected_daily_cost: float = 0.0
    actual_daily_cost: float = 0.0
    delta_dollars: float = 0.0
    delta_percent: float = 0.0
    z_score: float = 0.0
    lookback_days: int = 30


class TriageEvidence(BaseModel):
    deploy_correlation: list[str] = Field(default_factory=list)
    usage_correlation: list[str] = Field(default_factory=list)
    infra_correlation: list[str] = Field(default_factory=list)

    # finops primitives
    ri_coverage_delta: Optional[float] = None
    sp_coverage_delta: Optional[float] = None
    credits_delta: Optional[float] = None
    refunds_delta: Optional[float] = None
    fees_delta: Optional[float] = None
    marketplace_delta: Optional[float] = None
    data_transfer_delta: Optional[float] = None
    k8s_namespace_deltas: dict[str, float] = Field(default_factory=dict)


class TriageResult(BaseModel):
    category: AnomalyCategory = AnomalyCategory.unknown
    severity: AnomalySeverity = AnomalySeverity.medium
    confidence: float = 0.0
    summary: str = ""
    evidence: TriageEvidence = Field(default_factory=TriageEvidence)


class RecommendedAction(BaseModel):
    action_id: str = Field(default_factory=lambda: str(uuid.uuid4())[:8])
    description: str
    action_type: str
    risk_level: ActionRiskLevel
    estimated_savings_monthly: float = 0.0
    target_resource: str = ""
    parameters: dict[str, Any] = Field(default_factory=dict)
    rollback_procedure: str


class AnalysisResult(BaseModel):
    root_cause_narrative: str
    affected_resources: list[str] = Field(default_factory=list)
    recommended_actions: list[RecommendedAction] = Field(default_factory=list)
    estimated_monthly_savings: float = 0.0
    confidence: float = 0.0


class ExecutionResult(BaseModel):
    action_id: str
    executed_at: str
    success: bool
    details: str
    rollback_available: bool = True
    pre_action_snapshot: dict[str, Any] = Field(default_factory=dict)
    post_action_snapshot: dict[str, Any] = Field(default_factory=dict)


class VerificationResult(BaseModel):
    verified_at: str
    cost_reduction_observed: bool
    observed_savings_daily: float
    service_health_ok: bool
    health_check_details: str
    recommendation: Literal["close", "rollback", "escalate", "monitor"]


class TenantContext(BaseModel):
    tenant_id: str
    aws_management_account_id: str = ""
    default_region: str = "us-east-1"
    # in prod: sts role arn(s) per account
    iam_role_arn: str = ""
    # kubecost
    kubecost_base_url: str = ""


class FinOpsState(BaseModel):
    workflow_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    started_at: str = Field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

    tenant: TenantContext

    anomaly: Optional[CostAnomaly] = None
    triage: Optional[TriageResult] = None
    analysis: Optional[AnalysisResult] = None

    approval: ApprovalStatus = ApprovalStatus.pending
    approval_details: str = ""

    executions: list[ExecutionResult] = Field(default_factory=list)
    verification: Optional[VerificationResult] = None

    # control
    current_phase: str = "watcher"
    should_terminate: bool = False
    error: Optional[str] = None
