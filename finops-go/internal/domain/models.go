package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func shortID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// CostAnomaly represents a detected cost anomaly.
type CostAnomaly struct {
	AnomalyID  string `json:"anomaly_id"`
	DetectedAt string `json:"detected_at"`

	Service   string `json:"service"`
	AccountID string `json:"account_id"`
	Region    string `json:"region"`
	Team      string `json:"team"`

	ExpectedDailyCost float64 `json:"expected_daily_cost"`
	ActualDailyCost   float64 `json:"actual_daily_cost"`
	DeltaDollars      float64 `json:"delta_dollars"`
	DeltaPercent      float64 `json:"delta_percent"`
	ZScore            float64 `json:"z_score"`
	LookbackDays      int     `json:"lookback_days"`
}

// NewCostAnomaly creates a CostAnomaly with generated defaults.
func NewCostAnomaly() CostAnomaly {
	return CostAnomaly{
		AnomalyID:    shortID(),
		DetectedAt:   nowUTC(),
		LookbackDays: 30,
	}
}

// WasteFinding represents a single resource waste finding from aws-doctor.
type WasteFinding struct {
	ResourceType            string  `json:"resource_type"`
	ResourceID              string  `json:"resource_id"`
	ResourceARN             string  `json:"resource_arn"`
	Reason                  string  `json:"reason"`
	EstimatedMonthlySavings float64 `json:"estimated_monthly_savings"`
	Region                  string  `json:"region"`
}

// TriageEvidence holds correlation evidence collected during triage.
type TriageEvidence struct {
	DeployCorrelation []string `json:"deploy_correlation"`
	UsageCorrelation  []string `json:"usage_correlation"`
	InfraCorrelation  []string `json:"infra_correlation"`

	RICoverageDelta    *float64           `json:"ri_coverage_delta"`
	SPCoverageDelta    *float64           `json:"sp_coverage_delta"`
	CreditsDelta       *float64           `json:"credits_delta"`
	RefundsDelta       *float64           `json:"refunds_delta"`
	FeesDelta          *float64           `json:"fees_delta"`
	MarketplaceDelta   *float64           `json:"marketplace_delta"`
	DataTransferDelta  *float64           `json:"data_transfer_delta"`
	K8sNamespaceDeltas map[string]float64 `json:"k8s_namespace_deltas"`

	WasteFindings    []WasteFinding `json:"waste_findings,omitempty"`
	WasteSavings     *float64       `json:"waste_savings,omitempty"`
	TrendVelocityPct *float64       `json:"trend_velocity_pct,omitempty"`
	TrendDirection   string         `json:"trend_direction,omitempty"`
}

// TriageResult is the output of the triage classifier.
type TriageResult struct {
	Category   AnomalyCategory `json:"category"`
	Severity   AnomalySeverity `json:"severity"`
	Confidence float64         `json:"confidence"`
	Summary    string          `json:"summary"`
	Evidence   TriageEvidence  `json:"evidence"`
}

// RecommendedAction is a proposed action from the analyst.
type RecommendedAction struct {
	ActionID                string          `json:"action_id"`
	Description             string          `json:"description" validate:"required"`
	ActionType              string          `json:"action_type" validate:"required"`
	RiskLevel               ActionRiskLevel `json:"risk_level" validate:"required"`
	EstimatedSavingsMonthly float64         `json:"estimated_savings_monthly"`
	TargetResource          string          `json:"target_resource"`
	Parameters              map[string]any  `json:"parameters"`
	RollbackProcedure       string          `json:"rollback_procedure" validate:"required"`
}

// NewRecommendedAction creates a RecommendedAction with a generated ID.
func NewRecommendedAction(description, actionType string, riskLevel ActionRiskLevel, rollbackProcedure string) RecommendedAction {
	return RecommendedAction{
		ActionID:          shortID(),
		Description:       description,
		ActionType:        actionType,
		RiskLevel:         riskLevel,
		RollbackProcedure: rollbackProcedure,
	}
}

// AnalysisResult is the output of the analysis planner.
type AnalysisResult struct {
	RootCauseNarrative      string              `json:"root_cause_narrative"`
	AffectedResources       []string            `json:"affected_resources"`
	RecommendedActions      []RecommendedAction `json:"recommended_actions"`
	EstimatedMonthlySavings float64             `json:"estimated_monthly_savings"`
	Confidence              float64             `json:"confidence"`
}

// ExecutionResult records the outcome of executing an action.
type ExecutionResult struct {
	ActionID           string         `json:"action_id"`
	ExecutedAt         string         `json:"executed_at"`
	Success            bool           `json:"success"`
	Details            string         `json:"details"`
	RollbackAvailable  bool           `json:"rollback_available"`
	PreActionSnapshot  map[string]any `json:"pre_action_snapshot"`
	PostActionSnapshot map[string]any `json:"post_action_snapshot"`
}

// VerificationResult records the outcome of post-execution verification.
type VerificationResult struct {
	VerifiedAt            string                     `json:"verified_at"`
	CostReductionObserved bool                       `json:"cost_reduction_observed"`
	ObservedSavingsDaily  float64                    `json:"observed_savings_daily"`
	ServiceHealthOK       bool                       `json:"service_health_ok"`
	HealthCheckDetails    string                     `json:"health_check_details"`
	Recommendation        VerificationRecommendation `json:"recommendation"`
}

// TenantContext identifies a tenant and their cloud accounts.
type TenantContext struct {
	TenantID               string `json:"tenant_id" validate:"required"`
	AWSManagementAccountID string `json:"aws_management_account_id"`
	DefaultRegion          string `json:"default_region"`
	IAMRoleARN             string `json:"iam_role_arn"`
	KubecostBaseURL        string `json:"kubecost_base_url"`
}

// NewTenantContext creates a TenantContext with sensible defaults.
func NewTenantContext(tenantID string) TenantContext {
	return TenantContext{
		TenantID:      tenantID,
		DefaultRegion: "us-east-1",
	}
}

// FinOpsState is the top-level workflow state, analogous to the LangGraph state schema.
type FinOpsState struct {
	WorkflowID string `json:"workflow_id"`
	StartedAt  string `json:"started_at"`

	Tenant   TenantContext   `json:"tenant"`
	Anomaly  *CostAnomaly    `json:"anomaly"`
	Triage   *TriageResult   `json:"triage"`
	Analysis *AnalysisResult `json:"analysis"`

	Approval        ApprovalStatus `json:"approval"`
	ApprovalDetails string         `json:"approval_details"`

	Executions   []ExecutionResult   `json:"executions"`
	Verification *VerificationResult `json:"verification"`

	CurrentPhase    string  `json:"current_phase"`
	ShouldTerminate bool    `json:"should_terminate"`
	Error           *string `json:"error"`
}

// NewFinOpsState creates a FinOpsState with generated defaults.
func NewFinOpsState(tenant TenantContext) FinOpsState {
	return FinOpsState{
		WorkflowID:   newUUID(),
		StartedAt:    nowUTC(),
		Tenant:       tenant,
		Approval:     ApprovalPending,
		CurrentPhase: "watcher",
	}
}
