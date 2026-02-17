package uischema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/uischema"
)

func baseState() domain.FinOpsState {
	s := domain.NewFinOpsState(domain.NewTenantContext("t1"))
	s.Anomaly = &domain.CostAnomaly{
		AnomalyID:    "anom-1",
		Service:      "EC2",
		AccountID:    "123456789012",
		DeltaDollars: 750,
		DeltaPercent: 25,
		DetectedAt:   "2026-02-17T00:00:00Z",
	}
	return s
}

func TestBuild_WatcherPhase_OnlySummary(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "watcher"

	schema := uischema.Build(state)
	assert.Equal(t, "v1", schema.Version)
	assert.Equal(t, "watcher", schema.Phase)
	require.Len(t, schema.Components, 1)
	assert.Equal(t, uischema.ComponentAnomalySummary, schema.Components[0].Type)
	assert.Empty(t, schema.Actions)
}

func TestBuild_AfterTriage_CommitmentDrift(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "triage"
	ri := -5.0
	state.Triage = &domain.TriageResult{
		Category:   domain.CategoryCommitmentCoverageDrift,
		Severity:   domain.SeverityHigh,
		Confidence: 0.9,
		Summary:    "RI coverage dropped",
		Evidence:   domain.TriageEvidence{RICoverageDelta: &ri},
	}

	schema := uischema.Build(state)
	require.Len(t, schema.Components, 3)
	assert.Equal(t, uischema.ComponentAnomalySummary, schema.Components[0].Type)
	assert.Equal(t, uischema.ComponentTriageCard, schema.Components[1].Type)
	assert.Equal(t, uischema.ComponentCommitmentDrift, schema.Components[2].Type)
	assert.Equal(t, -5.0, schema.Components[2].Data["ri_coverage_delta"])
}

func TestBuild_AfterTriage_CreditsRefundsFees(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "triage"
	credits := -100.0
	state.Triage = &domain.TriageResult{
		Category: domain.CategoryCreditsRefundsFees,
		Severity: domain.SeverityMedium,
		Evidence: domain.TriageEvidence{CreditsDelta: &credits},
	}

	schema := uischema.Build(state)
	require.Len(t, schema.Components, 3)
	assert.Equal(t, uischema.ComponentCreditBreakdown, schema.Components[2].Type)
	assert.Equal(t, "Credits / Refunds / Fees", schema.Components[2].Title)
}

func TestBuild_AfterTriage_K8sCostShift(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "triage"
	state.Triage = &domain.TriageResult{
		Category: domain.CategoryK8sCostShift,
		Severity: domain.SeverityMedium,
		Evidence: domain.TriageEvidence{
			K8sNamespaceDeltas: map[string]float64{"prod": 200, "staging": -50},
		},
	}

	schema := uischema.Build(state)
	assert.Equal(t, uischema.ComponentK8sNamespaceDeltas, schema.Components[2].Type)
}

func TestBuild_AfterTriage_DeployRelated(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "triage"
	state.Triage = &domain.TriageResult{
		Category: domain.CategoryDeployRelated,
		Severity: domain.SeverityMedium,
		Evidence: domain.TriageEvidence{
			DeployCorrelation: []string{"deploy-abc123"},
		},
	}

	schema := uischema.Build(state)
	assert.Equal(t, uischema.ComponentDeployCorrelation, schema.Components[2].Type)
}

func TestBuild_AfterTriage_DataTransfer(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "triage"
	dt := 300.0
	state.Triage = &domain.TriageResult{
		Category: domain.CategoryDataTransfer,
		Severity: domain.SeverityMedium,
		Evidence: domain.TriageEvidence{DataTransferDelta: &dt},
	}

	schema := uischema.Build(state)
	assert.Equal(t, uischema.ComponentDataTransferSpike, schema.Components[2].Type)
}

func TestBuild_AfterTriage_Marketplace(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "triage"
	mp := 500.0
	state.Triage = &domain.TriageResult{
		Category: domain.CategoryMarketplace,
		Severity: domain.SeverityLow,
		Evidence: domain.TriageEvidence{MarketplaceDelta: &mp},
	}

	schema := uischema.Build(state)
	assert.Equal(t, uischema.ComponentCreditBreakdown, schema.Components[2].Type)
	assert.Equal(t, "Marketplace Charges", schema.Components[2].Title)
}

func TestBuild_AfterTriage_ExpectedGrowth(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "triage"
	state.Triage = &domain.TriageResult{
		Category: domain.CategoryExpectedGrowth,
		Severity: domain.SeverityLow,
		Evidence: domain.TriageEvidence{
			UsageCorrelation: []string{"CPU utilization increased 30%"},
		},
	}

	schema := uischema.Build(state)
	assert.Equal(t, uischema.ComponentCostTimeseries, schema.Components[2].Type)
}

func TestBuild_AfterTriage_Unknown(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "triage"
	state.Triage = &domain.TriageResult{
		Category: domain.CategoryUnknown,
		Severity: domain.SeverityLow,
	}

	schema := uischema.Build(state)
	assert.Equal(t, uischema.ComponentEvidencePanel, schema.Components[2].Type)
}

func TestBuild_AfterAnalysis_ActionPlanAndEditors(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "analyst"
	state.Approval = domain.ApprovalAutoApproved // not pending, so no approval_queue
	state.Triage = &domain.TriageResult{Category: domain.CategoryConfigDrift, Severity: domain.SeverityMedium}
	state.Analysis = &domain.AnalysisResult{
		RootCauseNarrative: "config drift detected",
		RecommendedActions: []domain.RecommendedAction{
			domain.NewRecommendedAction("resize instance", "modify_instance", domain.RiskMedium, "revert"),
			domain.NewRecommendedAction("create alert", "create_budget_alert", domain.RiskLow, "disable"),
		},
	}

	schema := uischema.Build(state)
	// summary + triage + evidence + action_plan + 2 editors = 6
	require.Len(t, schema.Components, 6)
	assert.Equal(t, uischema.ComponentActionPlan, schema.Components[3].Type)
	assert.Equal(t, uischema.ComponentActionEditor, schema.Components[4].Type)
	assert.Equal(t, uischema.ComponentActionEditor, schema.Components[5].Type)
}

func TestBuild_HILGatePending_ApproveAndDenyActions(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "hil_gate"
	state.Triage = &domain.TriageResult{Category: domain.CategoryConfigDrift, Severity: domain.SeverityMedium}
	state.Analysis = &domain.AnalysisResult{
		RecommendedActions: []domain.RecommendedAction{
			domain.NewRecommendedAction("resize", "modify_instance", domain.RiskMedium, "revert"),
		},
	}
	state.Approval = domain.ApprovalPending

	schema := uischema.Build(state)
	// summary + triage + evidence + action_plan + 1 editor + approval_queue = 6
	require.Len(t, schema.Components, 6)
	assert.Equal(t, uischema.ComponentApprovalQueue, schema.Components[5].Type)
	require.Len(t, schema.Actions, 2)
	assert.Equal(t, uischema.ActionApprove, schema.Actions[0].Type)
	assert.Equal(t, uischema.ActionDeny, schema.Actions[1].Type)
	// Medium risk: no confirm required.
	assert.Nil(t, schema.Actions[0].Confirm)
}

func TestBuild_HILGatePending_HighRiskConfirm(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "hil_gate"
	state.Triage = &domain.TriageResult{Category: domain.CategoryConfigDrift, Severity: domain.SeverityHigh}
	state.Analysis = &domain.AnalysisResult{
		RecommendedActions: []domain.RecommendedAction{
			domain.NewRecommendedAction("terminate instances", "terminate", domain.RiskHigh, "relaunch"),
		},
	}
	state.Approval = domain.ApprovalPending

	schema := uischema.Build(state)
	require.Len(t, schema.Actions, 2)
	require.NotNil(t, schema.Actions[0].Confirm)
	assert.True(t, schema.Actions[0].Confirm.Required)
}

func TestBuild_AfterExecution(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "executor"
	state.Triage = &domain.TriageResult{Category: domain.CategoryDeployRelated, Severity: domain.SeverityMedium}
	state.Analysis = &domain.AnalysisResult{
		RecommendedActions: []domain.RecommendedAction{
			domain.NewRecommendedAction("alert", "create_budget_alert", domain.RiskLow, "disable"),
		},
	}
	state.Approval = domain.ApprovalAutoApproved
	state.Executions = []domain.ExecutionResult{
		{ActionID: "a1", Success: true},
	}

	schema := uischema.Build(state)
	// Find execution_results component.
	found := false
	for _, c := range schema.Components {
		if c.Type == uischema.ComponentExecutionResults {
			found = true
		}
	}
	assert.True(t, found, "expected execution_results component")
}

func TestBuild_AfterVerification_Rollback(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "completed"
	state.Triage = &domain.TriageResult{Category: domain.CategoryConfigDrift, Severity: domain.SeverityMedium}
	state.Analysis = &domain.AnalysisResult{
		RecommendedActions: []domain.RecommendedAction{
			domain.NewRecommendedAction("resize", "modify", domain.RiskMedium, "revert"),
		},
	}
	state.Approval = domain.ApprovalApproved
	state.Executions = []domain.ExecutionResult{{ActionID: "a1", Success: true}}
	state.Verification = &domain.VerificationResult{
		Recommendation:  domain.RecommendRollback,
		ServiceHealthOK: false,
	}

	schema := uischema.Build(state)
	// Should have rollback action.
	require.Len(t, schema.Actions, 1)
	assert.Equal(t, uischema.ActionRollback, schema.Actions[0].Type)
	assert.NotNil(t, schema.Actions[0].Confirm)
}

func TestBuild_AfterVerification_Escalate(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "completed"
	state.Triage = &domain.TriageResult{Category: domain.CategoryConfigDrift, Severity: domain.SeverityHigh}
	state.Analysis = &domain.AnalysisResult{
		RecommendedActions: []domain.RecommendedAction{
			domain.NewRecommendedAction("resize", "modify", domain.RiskMedium, "revert"),
		},
	}
	state.Approval = domain.ApprovalApproved
	state.Executions = []domain.ExecutionResult{{ActionID: "a1", Success: true}}
	state.Verification = &domain.VerificationResult{
		Recommendation: domain.RecommendEscalate,
	}

	schema := uischema.Build(state)
	require.Len(t, schema.Actions, 1)
	assert.Equal(t, uischema.ActionEscalate, schema.Actions[0].Type)
}

func TestBuild_NoAnomaly(t *testing.T) {
	state := domain.NewFinOpsState(domain.NewTenantContext("t1"))
	state.CurrentPhase = "watcher"
	state.Anomaly = nil

	schema := uischema.Build(state)
	assert.Empty(t, schema.Components)
}

func TestBuild_AfterVerification_Close(t *testing.T) {
	state := baseState()
	state.CurrentPhase = "completed"
	state.Triage = &domain.TriageResult{Category: domain.CategoryDeployRelated, Severity: domain.SeverityLow}
	state.Analysis = &domain.AnalysisResult{
		RecommendedActions: []domain.RecommendedAction{
			domain.NewRecommendedAction("alert", "create_budget_alert", domain.RiskLow, "disable"),
		},
	}
	state.Approval = domain.ApprovalAutoApproved
	state.Executions = []domain.ExecutionResult{{ActionID: "a1", Success: true}}
	state.Verification = &domain.VerificationResult{
		Recommendation:        domain.RecommendClose,
		CostReductionObserved: true,
		ServiceHealthOK:       true,
	}

	schema := uischema.Build(state)
	// Close: no rollback/escalate actions.
	assert.Empty(t, schema.Actions)
	// But verification dashboard should be present.
	found := false
	for _, c := range schema.Components {
		if c.Type == uischema.ComponentVerificationDashboard {
			found = true
		}
	}
	assert.True(t, found)
}
