from __future__ import annotations

from .models import VerificationResult
from .tools import CostTools, InfraTools, now_utc_iso


def verify(
    *,
    service: str,
    account_id: str,
    cost: CostTools,
    infra: InfraTools,
    window_start: str,
    window_end: str,
) -> VerificationResult:
    # stub health
    health_ok = True
    health_details = "stub: ok"

    # stub cost check: read from fixture and pretend delta is observed
    ts = cost.get_cost_timeseries(service, account_id, window_start, window_end)
    observed = float(ts.get("observed_savings_daily", 0.0))

    if not health_ok:
        return VerificationResult(
            verified_at=now_utc_iso(),
            cost_reduction_observed=False,
            observed_savings_daily=0.0,
            service_health_ok=False,
            health_check_details=health_details,
            recommendation="rollback",
        )

    if observed > 0:
        return VerificationResult(
            verified_at=now_utc_iso(),
            cost_reduction_observed=True,
            observed_savings_daily=observed,
            service_health_ok=True,
            health_check_details=health_details,
            recommendation="close",
        )

    return VerificationResult(
        verified_at=now_utc_iso(),
        cost_reduction_observed=False,
        observed_savings_daily=0.0,
        service_health_ok=True,
        health_check_details=health_details,
        recommendation="monitor",
    )
