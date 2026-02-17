from __future__ import annotations

import json
import os
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Optional


# tool interfaces are deterministic; implementations can be real (aws sdk) or stubs.


@dataclass
class CostExplorerPoint:
    date: str
    amount: float


class CostTools:
    """abstract interface for cost/billing primitives."""

    # --- coarse cost ---
    def get_cost_timeseries(
        self,
        service: str,
        account_id: str,
        start_date: str,
        end_date: str,
        group_by: Optional[list[tuple[str, str]]] = None,  # (type, key)
        metric: str = "UnblendedCost",
    ) -> dict[str, Any]:
        raise NotImplementedError

    # --- cur line items (athena) ---
    def get_cur_line_items(
        self,
        account_id: str,
        start_date: str,
        end_date: str,
        service: Optional[str] = None,
        where_sql: str = "",
        limit: int = 2000,
    ) -> list[dict[str, Any]]:
        raise NotImplementedError

    # --- commitments ---
    def get_ri_coverage(self, account_id: str, start_date: str, end_date: str) -> dict[str, Any]:
        raise NotImplementedError

    def get_ri_utilization(self, account_id: str, start_date: str, end_date: str) -> dict[str, Any]:
        raise NotImplementedError

    def get_sp_coverage(self, account_id: str, start_date: str, end_date: str) -> dict[str, Any]:
        raise NotImplementedError

    def get_sp_utilization(self, account_id: str, start_date: str, end_date: str) -> dict[str, Any]:
        raise NotImplementedError


class InfraTools:
    def recent_deploys(self, service: str, lookback_hours: int = 48) -> list[dict[str, Any]]:
        raise NotImplementedError

    def cloudwatch_metrics(self, resource_id: str, metric_name: str, namespace: str, lookback_hours: int = 24) -> dict[str, Any]:
        raise NotImplementedError

    def resource_tags(self, resource_arn: str) -> dict[str, str]:
        raise NotImplementedError


class KubeCostTools:
    def allocation(self, window: str, aggregate: str = "namespace") -> dict[str, Any]:
        raise NotImplementedError


# --- stub implementations ---
class StubCostTools(CostTools):
    """reads fixtures from tools/fixtures/*.json; good enough for tests + local runs."""

    def __init__(self, fixtures_dir: str):
        self.fixtures_dir = fixtures_dir

    def _load(self, name: str) -> Any:
        path = os.path.join(self.fixtures_dir, name)
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)

    def get_cost_timeseries(self, service: str, account_id: str, start_date: str, end_date: str, group_by=None, metric: str = "UnblendedCost") -> dict[str, Any]:
        return self._load("cost_timeseries.json")

    def get_cur_line_items(self, account_id: str, start_date: str, end_date: str, service: Optional[str] = None, where_sql: str = "", limit: int = 2000) -> list[dict[str, Any]]:
        return self._load("cur_line_items.json")

    def get_ri_coverage(self, account_id: str, start_date: str, end_date: str) -> dict[str, Any]:
        return self._load("ri_coverage.json")

    def get_ri_utilization(self, account_id: str, start_date: str, end_date: str) -> dict[str, Any]:
        return self._load("ri_utilization.json")

    def get_sp_coverage(self, account_id: str, start_date: str, end_date: str) -> dict[str, Any]:
        return self._load("sp_coverage.json")

    def get_sp_utilization(self, account_id: str, start_date: str, end_date: str) -> dict[str, Any]:
        return self._load("sp_utilization.json")


class StubInfraTools(InfraTools):
    def __init__(self, fixtures_dir: str):
        self.fixtures_dir = fixtures_dir

    def _load(self, name: str) -> Any:
        path = os.path.join(self.fixtures_dir, name)
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)

    def recent_deploys(self, service: str, lookback_hours: int = 48) -> list[dict[str, Any]]:
        return self._load("deploys.json")

    def cloudwatch_metrics(self, resource_id: str, metric_name: str, namespace: str, lookback_hours: int = 24) -> dict[str, Any]:
        return self._load("cloudwatch_metrics.json")

    def resource_tags(self, resource_arn: str) -> dict[str, str]:
        return self._load("resource_tags.json")


class StubKubeCostTools(KubeCostTools):
    def __init__(self, fixtures_dir: str):
        self.fixtures_dir = fixtures_dir

    def allocation(self, window: str, aggregate: str = "namespace") -> dict[str, Any]:
        path = os.path.join(self.fixtures_dir, "kubecost_allocation.json")
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)


def now_utc_iso() -> str:
    return datetime.now(timezone.utc).isoformat()
