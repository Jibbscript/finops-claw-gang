package activities_test

import (
	"context"
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/executor"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
)

func newTestActivities() *activities.Activities {
	dir := testutil.GoldenDir()
	cost := &testutil.StubCost{FixturesDir: dir}
	infra := &testutil.StubInfra{FixturesDir: dir}
	kube := &testutil.StubKubeCost{FixturesDir: dir}
	exec := executor.NewExecutor(infra)
	return &activities.Activities{
		Cost:     cost,
		Infra:    infra,
		KubeCost: kube,
		Executor: exec,
	}
}

func TestTriageAnomaly_HappyPath(t *testing.T) {
	a := newTestActivities()
	out, err := a.TriageAnomaly(context.Background(), activities.TriageInput{
		Anomaly: domain.CostAnomaly{
			AnomalyID:    "test-1",
			Service:      "EC2",
			AccountID:    "123456789012",
			DeltaDollars: 750,
			DeltaPercent: 25,
		},
		WindowStart: "2026-02-01",
		WindowEnd:   "2026-02-16",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Result.Category.Valid() {
		t.Errorf("invalid category: %q", out.Result.Category)
	}
	if out.Result.Confidence <= 0 || out.Result.Confidence > 1 {
		t.Errorf("confidence out of range: %f", out.Result.Confidence)
	}
}

func TestPlanActions_HappyPath(t *testing.T) {
	a := newTestActivities()
	out, err := a.PlanActions(context.Background(), activities.PlanActionsInput{
		AccountID:   "123456789012",
		Service:     "EC2",
		WindowStart: "2026-02-01",
		WindowEnd:   "2026-02-16",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Result.RecommendedActions) == 0 {
		t.Error("expected at least one recommended action")
	}
	if out.Result.RootCauseNarrative == "" {
		t.Error("expected non-empty narrative")
	}
}

func TestExecuteActions_HappyPath(t *testing.T) {
	a := newTestActivities()
	action := domain.NewRecommendedAction(
		"create budget alert",
		"create_budget_alert",
		domain.RiskLow,
		"disable alert",
	)
	action.TargetResource = "budget:EC2:123456789012"

	out, err := a.ExecuteActions(context.Background(), activities.ExecuteActionsInput{
		Approval: domain.ApprovalAutoApproved,
		Actions:  []domain.RecommendedAction{action},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if !out.Results[0].Success {
		t.Error("expected success=true")
	}
}

func TestExecuteActions_UnapprovedDenied(t *testing.T) {
	a := newTestActivities()
	action := domain.NewRecommendedAction(
		"something",
		"do_thing",
		domain.RiskLow,
		"undo thing",
	)
	_, err := a.ExecuteActions(context.Background(), activities.ExecuteActionsInput{
		Approval: domain.ApprovalPending,
		Actions:  []domain.RecommendedAction{action},
	})
	if err == nil {
		t.Fatal("expected error for unapproved execution")
	}
}

func TestVerifyOutcome_HappyPath(t *testing.T) {
	a := newTestActivities()
	out, err := a.VerifyOutcome(context.Background(), activities.VerifyOutcomeInput{
		Service:     "EC2",
		AccountID:   "123456789012",
		WindowStart: "2026-02-01",
		WindowEnd:   "2026-02-16",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Result.Recommendation.Valid() {
		t.Errorf("invalid recommendation: %q", out.Result.Recommendation)
	}
}

func TestNotifySlack_Stub(t *testing.T) {
	a := newTestActivities()
	err := a.NotifySlack(context.Background(), activities.NotifySlackInput{
		Channel: "#finops",
		Message: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateTicket_Stub(t *testing.T) {
	a := newTestActivities()
	out, err := a.CreateTicket(context.Background(), activities.CreateTicketInput{
		Title:       "Test ticket",
		Description: "test",
		Priority:    "low",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.TicketID == "" {
		t.Error("expected non-empty ticket ID")
	}
}

func TestRunAWSDocWaste_NilAWSDoc(t *testing.T) {
	a := newTestActivities() // AWSDoc is nil by default
	_, err := a.RunAWSDocWaste(context.Background(), activities.AWSDocWasteInput{
		AccountID: "123456789012",
		Region:    "us-east-1",
	})
	if err == nil {
		t.Fatal("expected error when AWSDoc is nil")
	}
}

func TestRunAWSDocWaste_WithStub(t *testing.T) {
	dir := testutil.GoldenDir()
	a := newTestActivities()
	a.AWSDoc = &testutil.StubAWSDoctor{FixturesDir: dir}

	out, err := a.RunAWSDocWaste(context.Background(), activities.AWSDocWasteInput{
		AccountID: "123456789012",
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Findings) == 0 {
		t.Error("expected at least one finding from stub")
	}
	if out.TotalSavings <= 0 {
		t.Error("expected positive total savings from stub")
	}
}

func TestRunAWSDocTrend_Stub(t *testing.T) {
	a := newTestActivities()
	out, err := a.RunAWSDocTrend(context.Background(), activities.AWSDocTrendInput{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.TrendDirection != "stable" {
		t.Errorf("trend_direction = %q, want stable", out.TrendDirection)
	}
	if out.VelocityPct != 0 {
		t.Errorf("velocity_pct = %f, want 0", out.VelocityPct)
	}
}
