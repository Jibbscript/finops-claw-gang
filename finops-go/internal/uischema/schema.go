// Package uischema defines the typed UI contract emitted by the backend.
// The frontend renders dynamic components based on this schema -- it never
// decides what to show on its own.
package uischema

// UISchema is the top-level schema the backend emits for a workflow state.
type UISchema struct {
	Version    string      `json:"ui_schema_version"`
	WorkflowID string      `json:"workflow_id"`
	Phase      string      `json:"phase"`
	Components []Component `json:"components"`
	Actions    []Action    `json:"actions"`
}

// ComponentType identifies what React component to render.
type ComponentType string

const (
	ComponentAnomalySummary        ComponentType = "anomaly_summary"
	ComponentTriageCard            ComponentType = "triage_card"
	ComponentEvidencePanel         ComponentType = "evidence_panel"
	ComponentCostTimeseries        ComponentType = "cost_timeseries"
	ComponentCommitmentDrift       ComponentType = "commitment_drift"
	ComponentCreditBreakdown       ComponentType = "credit_breakdown"
	ComponentK8sNamespaceDeltas    ComponentType = "k8s_namespace_deltas"
	ComponentDeployCorrelation     ComponentType = "deploy_correlation"
	ComponentDataTransferSpike     ComponentType = "data_transfer_spike"
	ComponentActionPlan            ComponentType = "action_plan"
	ComponentApprovalQueue         ComponentType = "approval_queue"
	ComponentExecutionResults      ComponentType = "execution_results"
	ComponentVerificationDashboard ComponentType = "verification_dashboard"
	ComponentActionEditor          ComponentType = "action_editor"
)

// Visibility controls component rendering.
type Visibility string

const (
	VisibilityVisible   Visibility = "visible"
	VisibilityHidden    Visibility = "hidden"
	VisibilityCollapsed Visibility = "collapsed"
)

// Component is a single renderable UI element.
type Component struct {
	Type       ComponentType  `json:"type"`
	Title      string         `json:"title"`
	Priority   int            `json:"priority"`
	Visibility Visibility     `json:"visibility"`
	Data       map[string]any `json:"data,omitempty"`
}

// ActionUIType classifies the user-facing action.
type ActionUIType string

const (
	ActionApprove   ActionUIType = "approve"
	ActionDeny      ActionUIType = "deny"
	ActionRollback  ActionUIType = "rollback"
	ActionEscalate  ActionUIType = "escalate"
	ActionEditParam ActionUIType = "edit_param"
)

// ConfirmConfig describes confirmation requirements for high-risk actions.
type ConfirmConfig struct {
	Required        bool   `json:"required"`
	AcknowledgeText string `json:"acknowledge_text,omitempty"`
}

// Action is a user-triggerable operation from the UI.
type Action struct {
	Type    ActionUIType   `json:"type"`
	Label   string         `json:"label"`
	Confirm *ConfirmConfig `json:"confirm,omitempty"`
}
