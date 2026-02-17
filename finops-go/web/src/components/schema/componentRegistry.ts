import type { ComponentType as ReactComponentType } from "react";
import type { ComponentType, UIComponent } from "@/lib/types";
import { AnomalySummary } from "../anomaly/AnomalySummary";
import { TriageCard } from "../anomaly/TriageCard";
import { EvidencePanel } from "../anomaly/EvidencePanel";
import { ActionPlan } from "../anomaly/ActionPlan";
import { ApprovalQueue } from "../anomaly/ApprovalQueue";
import { ExecutionResults } from "../anomaly/ExecutionResults";
import { VerificationDashboard } from "../anomaly/VerificationDashboard";

// Maps ComponentType -> React component.
// Components not in this registry are silently skipped.
export const componentRegistry: Partial<
  Record<ComponentType, ReactComponentType<{ component: UIComponent }>>
> = {
  anomaly_summary: AnomalySummary,
  triage_card: TriageCard,
  evidence_panel: EvidencePanel,
  cost_timeseries: EvidencePanel,
  commitment_drift: EvidencePanel,
  credit_breakdown: EvidencePanel,
  k8s_namespace_deltas: EvidencePanel,
  deploy_correlation: EvidencePanel,
  data_transfer_spike: EvidencePanel,
  action_plan: ActionPlan,
  approval_queue: ApprovalQueue,
  execution_results: ExecutionResults,
  verification_dashboard: VerificationDashboard,
  action_editor: ActionPlan,
};
