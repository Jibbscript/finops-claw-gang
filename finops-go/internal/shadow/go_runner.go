package shadow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/finops-claw-gang/finops-go/internal/analysis"
	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/policy"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
	"github.com/finops-claw-gang/finops-go/internal/triage"
)

// GoRunner invokes the Go triage/analysis/policy pipeline directly using stub fixtures.
type GoRunner struct {
	FixturesDir string
}

// Run executes the Go pipeline on the given fixtures and returns JSON output.
// The output has the same top-level keys as the Python CLI --json-output:
// {"triage": {...}, "analysis": {...}, "approval": {...}}.
func (r *GoRunner) Run(ctx context.Context, service string, delta float64) ([]byte, error) {
	cost := &testutil.StubCost{FixturesDir: r.FixturesDir}
	infra := &testutil.StubInfra{FixturesDir: r.FixturesDir}
	kubecost := &testutil.StubKubeCost{FixturesDir: r.FixturesDir}
	waste := &testutil.StubAWSDoctor{FixturesDir: r.FixturesDir}

	anomaly := domain.CostAnomaly{
		AnomalyID:         "shadow-run",
		Service:           service,
		AccountID:         "123456789012",
		Region:            "us-east-1",
		DeltaDollars:      delta,
		DeltaPercent:      25.0,
		ExpectedDailyCost: delta * 3,
		ActualDailyCost:   delta*3 + delta,
		LookbackDays:      30,
	}

	triageResult, err := triage.Triage(ctx, anomaly, cost, infra, kubecost, waste, "", "")
	if err != nil {
		return nil, fmt.Errorf("triage: %w", err)
	}

	analysisResult, err := analysis.AnalyzeAndRecommend(
		anomaly.AccountID, anomaly.Service, "2026-02-01", "2026-02-16", cost,
	)
	if err != nil {
		return nil, fmt.Errorf("analysis: %w", err)
	}

	pe := policy.NewPolicyEngine()
	decision := pe.Decide(analysisResult.RecommendedActions)

	output := map[string]any{
		"triage":   triageResult,
		"analysis": analysisResult,
		"approval": map[string]any{
			"status":  decision.Approval,
			"details": decision.Details,
		},
	}

	return json.MarshalIndent(output, "", "  ")
}
