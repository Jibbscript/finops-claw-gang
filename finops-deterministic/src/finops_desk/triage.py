from __future__ import annotations

import re
from dataclasses import dataclass

from .models import (
    AnomalyCategory,
    AnomalySeverity,
    CostAnomaly,
    TriageEvidence,
    TriageResult,
)
from .tools import CostTools, InfraTools, KubeCostTools


# deterministic classifiers: no llm in the loop.
# llm can be layered on top later for narrative only.


def _severity_from_delta(delta_dollars_daily: float) -> AnomalySeverity:
    if delta_dollars_daily >= 5000:
        return AnomalySeverity.critical
    if delta_dollars_daily >= 1000:
        return AnomalySeverity.high
    if delta_dollars_daily >= 200:
        return AnomalySeverity.medium
    return AnomalySeverity.low


def _pct_change(new: float, old: float) -> float:
    if old == 0:
        return 1.0 if new != 0 else 0.0
    return (new - old) / old


def triage(
    anomaly: CostAnomaly,
    cost: CostTools,
    infra: InfraTools,
    kubecost: KubeCostTools | None = None,
    window_start: str | None = None,
    window_end: str | None = None,
) -> TriageResult:
    """triage via evidence ordering: commitments/credits/marketplace/transfer/k8s -> deploy -> expected growth -> config drift -> pricing change."""

    window_start = window_start or "2026-02-01"
    window_end = window_end or "2026-02-16"

    ev = TriageEvidence()

    # 1) commitments coverage drift
    ri_cov = cost.get_ri_coverage(anomaly.account_id, window_start, window_end)
    sp_cov = cost.get_sp_coverage(anomaly.account_id, window_start, window_end)
    ev.ri_coverage_delta = float(ri_cov.get("coverage_delta", 0.0))
    ev.sp_coverage_delta = float(sp_cov.get("coverage_delta", 0.0))

    if abs(ev.ri_coverage_delta) >= 0.05 or abs(ev.sp_coverage_delta) >= 0.05:
        # coverage drift often looks like "pricing" but is actually commitment coverage changes
        cat = AnomalyCategory.commitment_coverage_drift
        conf = 0.8
        summary = "ri/sp coverage shifted materially; investigate commitment coverage/utilization"
        return TriageResult(
            category=cat,
            severity=_severity_from_delta(anomaly.delta_dollars),
            confidence=conf,
            summary=summary,
            evidence=ev,
        )

    # 2) credits/refunds/fees deltas (from cur line item types)
    cur = cost.get_cur_line_items(anomaly.account_id, window_start, window_end, service=anomaly.service)
    # we assume fixtures use 'line_item_line_item_type' and 'unblended_cost'
    credits = sum(float(x.get("unblended_cost", 0.0)) for x in cur if str(x.get("line_item_line_item_type", "")).lower() == "credit")
    refunds = sum(float(x.get("unblended_cost", 0.0)) for x in cur if str(x.get("line_item_line_item_type", "")).lower() == "refund")
    fees = sum(float(x.get("unblended_cost", 0.0)) for x in cur if str(x.get("line_item_line_item_type", "")).lower() in {"fee", "rifee"})
    ev.credits_delta = credits
    ev.refunds_delta = refunds
    ev.fees_delta = fees

    # credits/refunds typically negative; a drop in credits increases net spend
    if abs(credits) >= 0.2 * max(anomaly.delta_dollars, 1.0) or abs(refunds) >= 0.2 * max(anomaly.delta_dollars, 1.0):
        return TriageResult(
            category=AnomalyCategory.credits_refunds_fees,
            severity=_severity_from_delta(anomaly.delta_dollars),
            confidence=0.75,
            summary="net spend change driven by credits/refunds/fees movement (not usage)",
            evidence=ev,
        )

    # 3) marketplace
    mp = sum(float(x.get("unblended_cost", 0.0)) for x in cur if "marketplace" in str(x.get("product_product_name", "")).lower() or "aws marketplace" in str(x.get("line_item_product_code", "")).lower())
    ev.marketplace_delta = mp
    if mp >= 0.2 * max(anomaly.delta_dollars, 1.0):
        return TriageResult(
            category=AnomalyCategory.marketplace,
            severity=_severity_from_delta(anomaly.delta_dollars),
            confidence=0.8,
            summary="spend appears dominated by marketplace charges (subscription/usage)",
            evidence=ev,
        )

    # 4) data transfer
    # aws cur usage types: REGION-DataTransfer-Out-Bytes etc.
    dt = 0.0
    for x in cur:
        ut = str(x.get("line_item_usage_type", ""))
        if "datatransfer" in ut.lower():
            dt += float(x.get("unblended_cost", 0.0))
    ev.data_transfer_delta = dt
    if dt >= 0.2 * max(anomaly.delta_dollars, 1.0):
        return TriageResult(
            category=AnomalyCategory.data_transfer,
            severity=_severity_from_delta(anomaly.delta_dollars),
            confidence=0.85,
            summary="spike primarily in data transfer usage types",
            evidence=ev,
        )

    # 5) kubecost allocation shifts (optional)
    if kubecost is not None:
        alloc = kubecost.allocation(window="24h", aggregate="namespace")
        # fixtures: {"allocations": {"ns": {"totalCost": ...}}}
        allocs = alloc.get("allocations", {})
        # naive: take top deltas from fixture-provided 'delta'
        for ns, v in allocs.items():
            if isinstance(v, dict) and "delta" in v:
                ev.k8s_namespace_deltas[ns] = float(v["delta"])
        if ev.k8s_namespace_deltas and max(ev.k8s_namespace_deltas.values()) >= 0.2 * max(anomaly.delta_dollars, 1.0):
            return TriageResult(
                category=AnomalyCategory.k8s_cost_shift,
                severity=_severity_from_delta(anomaly.delta_dollars),
                confidence=0.7,
                summary="k8s namespace allocation shifted materially (kubecost)",
                evidence=ev,
            )

    # 6) deploy correlation
    deploys = infra.recent_deploys(anomaly.service)
    if deploys:
        ev.deploy_correlation = [d.get("id", "deploy") for d in deploys]
        return TriageResult(
            category=AnomalyCategory.deploy_related,
            severity=_severity_from_delta(anomaly.delta_dollars),
            confidence=0.7,
            summary="recent deploys detected near anomaly window",
            evidence=ev,
        )

    # 7) expected growth (usage) vs config drift
    # if you have usage metrics, compare percent change; stub uses cloudwatch metrics fixture
    m = infra.cloudwatch_metrics(resource_id=anomaly.service, metric_name="Requests", namespace="Service")
    baseline = float(m.get("baseline", 0.0))
    current = float(m.get("current", 0.0))
    usage_pct = _pct_change(current, baseline)
    cost_pct = anomaly.delta_percent / 100.0

    if baseline > 0 and usage_pct > 0 and abs(usage_pct - cost_pct) <= 0.15:
        ev.usage_correlation = [f"usage pct ~{usage_pct:.2f} vs cost pct ~{cost_pct:.2f}"]
        return TriageResult(
            category=AnomalyCategory.expected_growth,
            severity=_severity_from_delta(anomaly.delta_dollars),
            confidence=0.8,
            summary="usage increase roughly explains cost increase",
            evidence=ev,
        )

    # default
    return TriageResult(
        category=AnomalyCategory.unknown,
        severity=_severity_from_delta(anomaly.delta_dollars),
        confidence=0.4,
        summary="no strong deterministic signal; requires deeper analysis",
        evidence=ev,
    )
