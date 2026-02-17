package domain_test

import (
	"context"
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/analysis"
	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/executor"
	"github.com/finops-claw-gang/finops-go/internal/policy"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
	"github.com/finops-claw-gang/finops-go/internal/triage"
	"github.com/finops-claw-gang/finops-go/internal/verifier"
)

// TestEndToEndPipeline wires all components with stubs and executes the
// full triage -> analysis -> policy -> executor -> verifier pipeline.
// This is the Go equivalent of the Python graph's auto-approve path.
func TestEndToEndPipeline(t *testing.T) {
	t.Parallel()
	goldenDir := testutil.GoldenDir()
	cost := &testutil.StubCost{FixturesDir: goldenDir}
	infra := &testutil.StubInfra{FixturesDir: goldenDir}
	kube := &testutil.StubKubeCost{FixturesDir: goldenDir}

	// 1. Create anomaly (same as Python test fixtures)
	anomaly := domain.CostAnomaly{
		AnomalyID:         "integ-001",
		DetectedAt:        "2026-02-16T00:00:00Z",
		Service:           "EC2",
		AccountID:         "123456789012",
		Region:            "us-east-1",
		Team:              "platform",
		ExpectedDailyCost: 2400.0,
		ActualDailyCost:   3150.0,
		DeltaDollars:      750.0,
		DeltaPercent:      31.25,
		ZScore:            3.2,
		LookbackDays:      30,
	}

	// 2. Triage
	triageResult, err := triage.Triage(context.Background(), anomaly, cost, infra, kube, nil, "", "")
	if err != nil {
		t.Fatalf("triage: %v", err)
	}
	if !triageResult.Category.Valid() {
		t.Fatalf("triage produced invalid category: %q", triageResult.Category)
	}
	t.Logf("triage: category=%s confidence=%.2f severity=%s",
		triageResult.Category, triageResult.Confidence, triageResult.Severity)

	// With current fixtures, expect data_transfer
	if triageResult.Category != domain.CategoryDataTransfer {
		t.Errorf("expected data_transfer, got %q", triageResult.Category)
	}

	// 3. Analysis
	analysisResult, err := analysis.AnalyzeAndRecommend(
		anomaly.AccountID, anomaly.Service, "2026-02-01", "2026-02-16", cost,
	)
	if err != nil {
		t.Fatalf("analysis: %v", err)
	}
	if len(analysisResult.RecommendedActions) == 0 {
		t.Fatal("analysis produced no recommended actions")
	}
	t.Logf("analysis: %d actions, first=%s risk=%s",
		len(analysisResult.RecommendedActions),
		analysisResult.RecommendedActions[0].ActionType,
		analysisResult.RecommendedActions[0].RiskLevel)

	// 4. Policy decision
	pe := policy.NewPolicyEngine()
	decision := pe.Decide(analysisResult.RecommendedActions)
	t.Logf("policy: approval=%s details=%q", decision.Approval, decision.Details)

	// Low-risk budget alert should be auto-approved
	if decision.Approval != domain.ApprovalAutoApproved {
		t.Errorf("expected auto_approved, got %q", decision.Approval)
	}

	// 5. Execute
	tagsByARN := make(map[string]map[string]string)
	for _, a := range analysisResult.RecommendedActions {
		if a.TargetResource != "" {
			tags, err := infra.ResourceTags(a.TargetResource)
			if err != nil {
				t.Fatalf("resource tags: %v", err)
			}
			tagsByARN[a.TargetResource] = tags
		}
	}

	exec := executor.NewExecutor(infra)
	execResults, err := exec.ExecuteActions(decision.Approval, analysisResult.RecommendedActions, tagsByARN)
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	if len(execResults) != len(analysisResult.RecommendedActions) {
		t.Fatalf("expected %d exec results, got %d",
			len(analysisResult.RecommendedActions), len(execResults))
	}
	for i, r := range execResults {
		t.Logf("exec[%d]: action=%s success=%v", i, r.ActionID, r.Success)
		if !r.Success {
			t.Errorf("execution %d failed: %s", i, r.Details)
		}
	}

	// 6. Verify
	verifyResult, err := verifier.Verify(anomaly.Service, anomaly.AccountID, cost, "2026-02-01", "2026-02-16")
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	t.Logf("verifier: recommendation=%s health_ok=%v savings=%.2f",
		verifyResult.Recommendation, verifyResult.ServiceHealthOK, verifyResult.ObservedSavingsDaily)

	// Fixture has observed_savings_daily=0.0 and health=ok, so expect "monitor"
	if verifyResult.Recommendation != domain.RecommendMonitor {
		t.Errorf("expected monitor, got %q", verifyResult.Recommendation)
	}
	if !verifyResult.ServiceHealthOK {
		t.Error("expected service health ok")
	}

	// 7. Validate final state assembly
	state := domain.NewFinOpsState(domain.NewTenantContext("t-001"))
	state.Anomaly = &anomaly
	state.Triage = &triageResult
	state.Analysis = &analysisResult
	state.Approval = decision.Approval
	state.ApprovalDetails = decision.Details
	state.Executions = execResults
	state.Verification = &verifyResult
	state.CurrentPhase = "verifier"

	if err := domain.ValidateFinOpsState(state); err != nil {
		t.Errorf("final state validation failed: %v", err)
	}
}

// TestEndToEndDeniedPath tests the path where policy denies execution.
func TestEndToEndDeniedPath(t *testing.T) {
	t.Parallel()
	pe := policy.NewPolicyEngine()

	// Empty actions -> denied
	decision := pe.Decide(nil)
	if decision.Approval != domain.ApprovalDenied {
		t.Errorf("expected denied for no actions, got %q", decision.Approval)
	}

	// Critical action -> denied
	criticalAction := domain.NewRecommendedAction(
		"terminate instances", "terminate", domain.RiskCritical, "n/a",
	)
	decision = pe.Decide([]domain.RecommendedAction{criticalAction})
	if decision.Approval != domain.ApprovalDenied {
		t.Errorf("expected denied for critical, got %q", decision.Approval)
	}
}

// TestEndToEndPendingPath tests the human-approval-required path.
func TestEndToEndPendingPath(t *testing.T) {
	t.Parallel()
	pe := policy.NewPolicyEngine()

	mediumAction := domain.NewRecommendedAction(
		"resize instance", "resize", domain.RiskMedium, "revert",
	)
	decision := pe.Decide([]domain.RecommendedAction{mediumAction})
	if decision.Approval != domain.ApprovalPending {
		t.Errorf("expected pending for medium risk, got %q", decision.Approval)
	}
}
