package analysis

import (
	"fmt"

	"github.com/finops-claw-gang/finops-go/internal/domain"
)

// AnalyzeAndRecommend reviews CUR line items for the given service and window,
// then returns an AnalysisResult with a narrative and recommended actions.
// This is a deterministic placeholder; in production the LLM may add narrative,
// but actions must still pass policy validation.
func AnalyzeAndRecommend(
	accountID, service, windowStart, windowEnd string,
	cost CostQuerier,
) (domain.AnalysisResult, error) {
	_, err := cost.GetCURLineItems(accountID, windowStart, windowEnd, service)
	if err != nil {
		return domain.AnalysisResult{}, fmt.Errorf("analysis: get CUR line items: %w", err)
	}

	narrative := fmt.Sprintf(
		"cur line items reviewed for %s %s..%s; further attribution required",
		service, windowStart, windowEnd,
	)

	action := domain.NewRecommendedAction(
		fmt.Sprintf("create/update budget alert for %s to catch recurrence", service),
		"create_budget_alert",
		domain.RiskLow,
		"disable alert / delete budget rule",
	)
	action.EstimatedSavingsMonthly = 0.0
	action.TargetResource = fmt.Sprintf("budget:%s:%s", service, accountID)
	action.Parameters = map[string]any{
		"amount":            0.0,
		"threshold_percent": 20.0,
	}

	return domain.AnalysisResult{
		RootCauseNarrative:      narrative,
		AffectedResources:       []string{},
		RecommendedActions:      []domain.RecommendedAction{action},
		EstimatedMonthlySavings: 0.0,
		Confidence:              0.4,
	}, nil
}
