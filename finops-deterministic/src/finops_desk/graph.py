from __future__ import annotations

from langgraph.graph import END, StateGraph
from langgraph.checkpoint.memory import MemorySaver
from langgraph.types import interrupt

from .models import FinOpsState, ApprovalStatus
from .policy import PolicyEngine
from .triage import triage
from .analysis import analyze_and_recommend
from .executor import DeterministicExecutor
from .verifier import verify
from .tools import CostTools, InfraTools, KubeCostTools


class Runtime:
    def __init__(self, cost: CostTools, infra: InfraTools, kubecost: KubeCostTools | None = None):
        self.cost = cost
        self.infra = infra
        self.kubecost = kubecost
        self.policy = PolicyEngine()
        self.executor = DeterministicExecutor(infra)


def watcher_node(state: FinOpsState) -> dict:
    # deterministic: watcher is external trigger in real life.
    if state.anomaly is None:
        state.should_terminate = True
    return {"current_phase": "watcher", "should_terminate": state.should_terminate}


def triager_node(runtime: Runtime):
    def _node(state: FinOpsState) -> dict:
        if not state.anomaly:
            return {"error": "missing anomaly", "current_phase": "triager"}
        state.triage = triage(state.anomaly, runtime.cost, runtime.infra, runtime.kubecost)
        return {"triage": state.triage, "current_phase": "triager"}

    return _node


def analyst_node(runtime: Runtime):
    def _node(state: FinOpsState) -> dict:
        if not state.anomaly:
            return {"error": "missing anomaly", "current_phase": "analyst"}
        # minimal deterministic analysis; in prod add llm narrative w/ strict schema
        state.analysis = analyze_and_recommend(
            account_id=state.anomaly.account_id,
            service=state.anomaly.service,
            window_start="2026-02-01",
            window_end="2026-02-16",
            cost=runtime.cost,
            infra=runtime.infra,
        )
        return {"analysis": state.analysis, "current_phase": "analyst"}

    return _node


def hil_gate_node(runtime: Runtime):
    def _node(state: FinOpsState) -> dict:
        actions = state.analysis.recommended_actions if state.analysis else []
        decision = runtime.policy.decide(actions)
        state.approval = decision.approval
        state.approval_details = decision.details

        # hard stop: if pending, interrupt + wait for external approval event
        if state.approval == ApprovalStatus.pending:
            payload = {
                "workflow_id": state.workflow_id,
                "summary": state.triage.summary if state.triage else "",
                "actions": [a.model_dump() for a in actions],
            }
            human = interrupt(payload)
            # human response should be {"approve": true/false, "by": "..."}
            if isinstance(human, dict) and human.get("approve") is True:
                state.approval = ApprovalStatus.approved
                state.approval_details = f"approved_by={human.get('by','unknown')}"
            else:
                state.approval = ApprovalStatus.denied
                state.approval_details = f"denied_by={human.get('by','unknown')}"

        return {
            "approval": state.approval,
            "approval_details": state.approval_details,
            "current_phase": "hil_gate",
        }

    return _node


def executor_node(runtime: Runtime):
    def _node(state: FinOpsState) -> dict:
        actions = state.analysis.recommended_actions if state.analysis else []
        # gather tags deterministically
        tags_by_arn = {a.target_resource: runtime.infra.resource_tags(a.target_resource) for a in actions if a.target_resource}
        state.executions = runtime.executor.execute_actions(state.approval, actions, tags_by_arn)
        return {"executions": state.executions, "current_phase": "executor"}

    return _node


def verifier_node(runtime: Runtime):
    def _node(state: FinOpsState) -> dict:
        if not state.anomaly:
            return {"error": "missing anomaly", "current_phase": "verifier"}
        state.verification = verify(
            service=state.anomaly.service,
            account_id=state.anomaly.account_id,
            cost=runtime.cost,
            infra=runtime.infra,
            window_start="2026-02-01",
            window_end="2026-02-16",
        )
        return {"verification": state.verification, "current_phase": "verifier"}

    return _node


def route_after_watcher(state: FinOpsState) -> str:
    if state.should_terminate:
        return "end"
    if state.error:
        return "end"
    return "triager"


def route_after_triager(state: FinOpsState) -> str:
    if state.error:
        return "end"
    # close expected growth high confidence
    if state.triage and state.triage.category.value == "expected_growth" and state.triage.confidence >= 0.85:
        return "end"
    return "analyst"


def route_after_analyst(state: FinOpsState) -> str:
    if state.error:
        return "end"
    if state.analysis and state.analysis.recommended_actions:
        return "hil_gate"
    return "end"


def route_after_hil(state: FinOpsState) -> str:
    if state.approval in (ApprovalStatus.approved, ApprovalStatus.auto_approved):
        return "executor"
    return "end"


def route_after_exec(state: FinOpsState) -> str:
    if state.error:
        return "end"
    return "verifier"


def route_after_verify(state: FinOpsState) -> str:
    if not state.verification:
        return "end"
    if state.verification.recommendation == "rollback":
        # rollback path omitted in skeleton
        return "end"
    return "end"


def build_app(runtime: Runtime):
    g = StateGraph(FinOpsState)

    g.add_node("watcher", watcher_node)
    g.add_node("triager", triager_node(runtime))
    g.add_node("analyst", analyst_node(runtime))
    g.add_node("hil_gate", hil_gate_node(runtime))
    g.add_node("executor", executor_node(runtime))
    g.add_node("verifier", verifier_node(runtime))

    g.set_entry_point("watcher")

    g.add_conditional_edges("watcher", route_after_watcher, {"triager": "triager", "end": END})
    g.add_conditional_edges("triager", route_after_triager, {"analyst": "analyst", "end": END})
    g.add_conditional_edges("analyst", route_after_analyst, {"hil_gate": "hil_gate", "end": END})
    g.add_conditional_edges("hil_gate", route_after_hil, {"executor": "executor", "end": END})
    g.add_conditional_edges("executor", route_after_exec, {"verifier": "verifier", "end": END})
    g.add_conditional_edges("verifier", route_after_verify, {"end": END})

    return g.compile(checkpointer=MemorySaver())
