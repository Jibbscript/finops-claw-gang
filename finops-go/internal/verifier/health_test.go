package verifier

import (
	"fmt"
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
)

func TestVerify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cost          CostChecker
		wantRec       domain.VerificationRecommendation
		wantReduction bool
		wantSavings   float64
		wantHealthOK  bool
	}{
		{
			name:          "monitor with fixtures (savings=0)",
			cost:          &testutil.StubCost{FixturesDir: testutil.GoldenDir()},
			wantRec:       domain.RecommendMonitor,
			wantReduction: false,
			wantSavings:   0.0,
			wantHealthOK:  true,
		},
		{
			name:          "close on positive savings",
			cost:          &mockCostChecker{timeseries: map[string]any{"observed_savings_daily": 50.0}},
			wantRec:       domain.RecommendClose,
			wantReduction: true,
			wantSavings:   50.0,
			wantHealthOK:  true,
		},
		{
			name:          "monitor on zero savings",
			cost:          &mockCostChecker{timeseries: map[string]any{"observed_savings_daily": 0.0}},
			wantRec:       domain.RecommendMonitor,
			wantReduction: false,
			wantSavings:   0.0,
			wantHealthOK:  true,
		},
		{
			name:          "monitor on negative savings",
			cost:          &mockCostChecker{timeseries: map[string]any{"observed_savings_daily": -10.0}},
			wantRec:       domain.RecommendMonitor,
			wantReduction: false,
			wantSavings:   0.0,
			wantHealthOK:  true,
		},
		{
			name:          "monitor on missing key",
			cost:          &mockCostChecker{timeseries: map[string]any{}},
			wantRec:       domain.RecommendMonitor,
			wantReduction: false,
			wantSavings:   0.0,
			wantHealthOK:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := Verify("EC2", "123456789012", tt.cost, "2026-02-01", "2026-02-16")
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if result.Recommendation != tt.wantRec {
				t.Errorf("recommendation = %q, want %q", result.Recommendation, tt.wantRec)
			}
			if result.CostReductionObserved != tt.wantReduction {
				t.Errorf("cost_reduction_observed = %v, want %v", result.CostReductionObserved, tt.wantReduction)
			}
			if result.ObservedSavingsDaily != tt.wantSavings {
				t.Errorf("observed_savings_daily = %f, want %f", result.ObservedSavingsDaily, tt.wantSavings)
			}
			if result.ServiceHealthOK != tt.wantHealthOK {
				t.Errorf("service_health_ok = %v, want %v", result.ServiceHealthOK, tt.wantHealthOK)
			}
		})
	}
}

func TestVerifyError(t *testing.T) {
	t.Parallel()
	cost := &mockCostChecker{err: errStub}
	_, err := Verify("EC2", "123456789012", cost, "2026-02-01", "2026-02-16")
	if err == nil {
		t.Error("expected error from failing CostChecker")
	}
}

type mockCostChecker struct {
	timeseries map[string]any
	err        error
}

var errStub = fmt.Errorf("stub error")

func (m *mockCostChecker) GetCostTimeseries(_, _, _, _ string) (map[string]any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.timeseries, nil
}
