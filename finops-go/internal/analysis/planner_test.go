package analysis

import (
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
)

func TestAnalyzeAndRecommend(t *testing.T) {
	t.Parallel()
	goldenDir := testutil.GoldenDir()
	cost := &testutil.StubCost{FixturesDir: goldenDir}

	result, err := AnalyzeAndRecommend("123456789012", "EC2", "2026-02-01", "2026-02-16", cost)
	if err != nil {
		t.Fatalf("AnalyzeAndRecommend: %v", err)
	}

	if len(result.RecommendedActions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.RecommendedActions))
	}

	action := result.RecommendedActions[0]

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{name: "action_type", check: func(t *testing.T) {
			t.Helper()
			if action.ActionType != "create_budget_alert" {
				t.Errorf("action_type = %q, want %q", action.ActionType, "create_budget_alert")
			}
		}},
		{name: "risk_level", check: func(t *testing.T) {
			t.Helper()
			if action.RiskLevel != domain.RiskLow {
				t.Errorf("risk_level = %q, want %q", action.RiskLevel, domain.RiskLow)
			}
		}},
		{name: "target_resource", check: func(t *testing.T) {
			t.Helper()
			if action.TargetResource != "budget:EC2:123456789012" {
				t.Errorf("target_resource = %q, want %q", action.TargetResource, "budget:EC2:123456789012")
			}
		}},
		{name: "confidence", check: func(t *testing.T) {
			t.Helper()
			if result.Confidence != 0.4 {
				t.Errorf("confidence = %f, want 0.4", result.Confidence)
			}
		}},
		{name: "action_id non-empty", check: func(t *testing.T) {
			t.Helper()
			if action.ActionID == "" {
				t.Error("expected non-empty action_id")
			}
		}},
		{name: "schema valid", check: func(t *testing.T) {
			t.Helper()
			if err := domain.ValidateRecommendedAction(action); err != nil {
				t.Errorf("action fails validation: %v", err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t)
		})
	}
}
