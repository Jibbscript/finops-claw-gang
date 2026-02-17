// TypeScript types mirroring Go domain types.

export interface CostAnomaly {
  anomaly_id: string;
  detected_at: string;
  service: string;
  account_id: string;
  region: string;
  team: string;
  expected_daily_cost: number;
  actual_daily_cost: number;
  delta_dollars: number;
  delta_percent: number;
  z_score: number;
  lookback_days: number;
}

export interface TriageEvidence {
  deploy_correlation: string[];
  usage_correlation: string[];
  infra_correlation: string[];
  ri_coverage_delta?: number;
  sp_coverage_delta?: number;
  credits_delta?: number;
  refunds_delta?: number;
  fees_delta?: number;
  marketplace_delta?: number;
  data_transfer_delta?: number;
  k8s_namespace_deltas?: Record<string, number>;
}

export interface TriageResult {
  category: string;
  severity: string;
  confidence: number;
  summary: string;
  evidence: TriageEvidence;
}

export interface RecommendedAction {
  action_id: string;
  description: string;
  action_type: string;
  risk_level: string;
  estimated_savings_monthly: number;
  target_resource: string;
  parameters: Record<string, unknown>;
  rollback_procedure: string;
}

export interface AnalysisResult {
  root_cause_narrative: string;
  affected_resources: string[];
  recommended_actions: RecommendedAction[];
  estimated_monthly_savings: number;
  confidence: number;
}

export interface ExecutionResult {
  action_id: string;
  executed_at: string;
  success: boolean;
  details: string;
  rollback_available: boolean;
}

export interface VerificationResult {
  verified_at: string;
  cost_reduction_observed: boolean;
  observed_savings_daily: number;
  service_health_ok: boolean;
  health_check_details: string;
  recommendation: string;
}

export interface FinOpsState {
  workflow_id: string;
  started_at: string;
  anomaly?: CostAnomaly;
  triage?: TriageResult;
  analysis?: AnalysisResult;
  approval: string;
  approval_details: string;
  executions: ExecutionResult[];
  verification?: VerificationResult;
  current_phase: string;
  should_terminate: boolean;
  error?: string;
}

export interface WorkflowResult {
  state: FinOpsState;
  reason: string;
}

export interface WorkflowSummary {
  workflow_id: string;
  run_id: string;
  status: string;
  start_time: string;
  close_time?: string;
  task_queue: string;
}

// UI Schema types

export type ComponentType =
  | "anomaly_summary"
  | "triage_card"
  | "evidence_panel"
  | "cost_timeseries"
  | "commitment_drift"
  | "credit_breakdown"
  | "k8s_namespace_deltas"
  | "deploy_correlation"
  | "data_transfer_spike"
  | "action_plan"
  | "approval_queue"
  | "execution_results"
  | "verification_dashboard"
  | "action_editor";

export interface UIComponent {
  type: ComponentType;
  title: string;
  priority: number;
  visibility: "visible" | "hidden" | "collapsed";
  data?: Record<string, unknown>;
}

export interface ConfirmConfig {
  required: boolean;
  acknowledge_text?: string;
}

export interface UIAction {
  type: "approve" | "deny" | "rollback" | "escalate" | "edit_param";
  label: string;
  confirm?: ConfirmConfig;
}

export interface UISchema {
  ui_schema_version: string;
  workflow_id: string;
  phase: string;
  components: UIComponent[];
  actions: UIAction[];
}
