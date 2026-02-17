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

// wasteActionType maps a waste finding resource type to an action type.
var wasteActionType = map[string]string{
	"EC2":          "terminate_instance",
	"EBS":          "delete_volume",
	"Snapshot":     "delete_snapshot",
	"ElasticIP":    "release_elastic_ip",
	"LoadBalancer": "delete_load_balancer",
	"AMI":          "deregister_ami",
}

// wasteRiskLevel maps a waste finding resource type to its risk level.
var wasteRiskLevel = map[string]domain.ActionRiskLevel{
	"EC2":          domain.RiskLow,    // stopped instances — low risk to terminate
	"EBS":          domain.RiskMedium, // volumes may contain data
	"Snapshot":     domain.RiskMedium, // snapshots may be the only backup
	"ElasticIP":    domain.RiskLow,
	"LoadBalancer": domain.RiskMedium,
	"AMI":          domain.RiskLow,
}

// wasteRollback maps a waste finding resource type to a rollback recipe.
var wasteRollback = map[string]string{
	"EC2":          "launch replacement from AMI or backup",
	"EBS":          "restore volume from snapshot",
	"Snapshot":     "no rollback — snapshot data is permanently lost",
	"ElasticIP":    "allocate new Elastic IP and update DNS",
	"LoadBalancer": "recreate load balancer with same configuration",
	"AMI":          "re-create AMI from running instance",
}

// AnalyzeWaste converts waste findings into candidate actions via templates.
// Every action is tied to a concrete resource ARN and produced with a rollback recipe.
func AnalyzeWaste(findings []domain.WasteFinding) domain.AnalysisResult {
	var (
		actions   []domain.RecommendedAction
		resources []string
		totalSav  float64
	)

	for _, f := range findings {
		actionType := wasteActionType[f.ResourceType]
		if actionType == "" {
			actionType = "review_resource"
		}
		risk := wasteRiskLevel[f.ResourceType]
		if risk == "" {
			risk = domain.RiskMedium
		}
		rollback := wasteRollback[f.ResourceType]
		if rollback == "" {
			rollback = "manual review required"
		}

		action := domain.NewRecommendedAction(
			fmt.Sprintf("%s %s (%s)", actionType, f.ResourceID, f.Reason),
			actionType,
			risk,
			rollback,
		)
		action.TargetResource = f.ResourceARN
		action.EstimatedSavingsMonthly = f.EstimatedMonthlySavings
		action.Parameters = map[string]any{
			"resource_type": f.ResourceType,
			"region":        f.Region,
		}

		actions = append(actions, action)
		resources = append(resources, f.ResourceARN)
		totalSav += f.EstimatedMonthlySavings
	}

	return domain.AnalysisResult{
		RootCauseNarrative:      fmt.Sprintf("aws-doctor identified %d waste findings across scanned resources", len(findings)),
		AffectedResources:       resources,
		RecommendedActions:      actions,
		EstimatedMonthlySavings: totalSav,
		Confidence:              0.85,
	}
}
