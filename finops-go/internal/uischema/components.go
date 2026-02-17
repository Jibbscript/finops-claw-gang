package uischema

import "github.com/finops-claw-gang/finops-go/internal/domain"

// anomalySummary builds the always-present anomaly overview component.
func anomalySummary(state domain.FinOpsState) Component {
	data := map[string]any{
		"service":       state.Anomaly.Service,
		"account_id":    state.Anomaly.AccountID,
		"delta_dollars": state.Anomaly.DeltaDollars,
		"delta_percent": state.Anomaly.DeltaPercent,
		"detected_at":   state.Anomaly.DetectedAt,
	}
	return Component{
		Type:       ComponentAnomalySummary,
		Title:      "Anomaly Summary",
		Priority:   0,
		Visibility: VisibilityVisible,
		Data:       data,
	}
}

// triageCard builds the triage classification card.
func triageCard(triage *domain.TriageResult) Component {
	return Component{
		Type:       ComponentTriageCard,
		Title:      "Triage Classification",
		Priority:   10,
		Visibility: VisibilityVisible,
		Data: map[string]any{
			"category":   string(triage.Category),
			"severity":   string(triage.Severity),
			"confidence": triage.Confidence,
			"summary":    triage.Summary,
		},
	}
}

// categoryEvidence returns the category-specific evidence component.
func categoryEvidence(triage *domain.TriageResult) Component {
	switch triage.Category {
	case domain.CategoryCommitmentCoverageDrift:
		return commitmentDrift(triage)
	case domain.CategoryCreditsRefundsFees:
		return creditBreakdown(triage, "Credits / Refunds / Fees")
	case domain.CategoryK8sCostShift:
		return k8sNamespaceDeltas(triage)
	case domain.CategoryDeployRelated:
		return deployCorrelation(triage)
	case domain.CategoryDataTransfer:
		return dataTransferSpike(triage)
	case domain.CategoryMarketplace:
		return creditBreakdown(triage, "Marketplace Charges")
	case domain.CategoryExpectedGrowth:
		return costTimeseries(triage)
	default:
		return evidencePanel(triage)
	}
}

func commitmentDrift(triage *domain.TriageResult) Component {
	data := map[string]any{
		"category": string(triage.Category),
	}
	if triage.Evidence.RICoverageDelta != nil {
		data["ri_coverage_delta"] = *triage.Evidence.RICoverageDelta
	}
	if triage.Evidence.SPCoverageDelta != nil {
		data["sp_coverage_delta"] = *triage.Evidence.SPCoverageDelta
	}
	return Component{
		Type:       ComponentCommitmentDrift,
		Title:      "Commitment Coverage Drift",
		Priority:   20,
		Visibility: VisibilityVisible,
		Data:       data,
	}
}

func creditBreakdown(triage *domain.TriageResult, title string) Component {
	data := map[string]any{
		"category": string(triage.Category),
	}
	if triage.Evidence.CreditsDelta != nil {
		data["credits_delta"] = *triage.Evidence.CreditsDelta
	}
	if triage.Evidence.RefundsDelta != nil {
		data["refunds_delta"] = *triage.Evidence.RefundsDelta
	}
	if triage.Evidence.FeesDelta != nil {
		data["fees_delta"] = *triage.Evidence.FeesDelta
	}
	if triage.Evidence.MarketplaceDelta != nil {
		data["marketplace_delta"] = *triage.Evidence.MarketplaceDelta
	}
	return Component{
		Type:       ComponentCreditBreakdown,
		Title:      title,
		Priority:   20,
		Visibility: VisibilityVisible,
		Data:       data,
	}
}

func k8sNamespaceDeltas(triage *domain.TriageResult) Component {
	data := map[string]any{
		"category":         string(triage.Category),
		"namespace_deltas": triage.Evidence.K8sNamespaceDeltas,
	}
	return Component{
		Type:       ComponentK8sNamespaceDeltas,
		Title:      "Kubernetes Namespace Cost Deltas",
		Priority:   20,
		Visibility: VisibilityVisible,
		Data:       data,
	}
}

func deployCorrelation(triage *domain.TriageResult) Component {
	return Component{
		Type:       ComponentDeployCorrelation,
		Title:      "Deploy Correlation",
		Priority:   20,
		Visibility: VisibilityVisible,
		Data: map[string]any{
			"category":           string(triage.Category),
			"deploy_correlation": triage.Evidence.DeployCorrelation,
		},
	}
}

func dataTransferSpike(triage *domain.TriageResult) Component {
	data := map[string]any{
		"category": string(triage.Category),
	}
	if triage.Evidence.DataTransferDelta != nil {
		data["data_transfer_delta"] = *triage.Evidence.DataTransferDelta
	}
	return Component{
		Type:       ComponentDataTransferSpike,
		Title:      "Data Transfer Spike",
		Priority:   20,
		Visibility: VisibilityVisible,
		Data:       data,
	}
}

func costTimeseries(triage *domain.TriageResult) Component {
	return Component{
		Type:       ComponentCostTimeseries,
		Title:      "Cost vs Usage Overlay",
		Priority:   20,
		Visibility: VisibilityVisible,
		Data: map[string]any{
			"category":          string(triage.Category),
			"usage_correlation": triage.Evidence.UsageCorrelation,
		},
	}
}

func evidencePanel(triage *domain.TriageResult) Component {
	return Component{
		Type:       ComponentEvidencePanel,
		Title:      "Evidence",
		Priority:   20,
		Visibility: VisibilityVisible,
		Data: map[string]any{
			"category":           string(triage.Category),
			"deploy_correlation": triage.Evidence.DeployCorrelation,
			"usage_correlation":  triage.Evidence.UsageCorrelation,
			"infra_correlation":  triage.Evidence.InfraCorrelation,
		},
	}
}

// actionPlan builds the action plan overview component.
func actionPlan(analysis *domain.AnalysisResult) Component {
	actions := make([]map[string]any, len(analysis.RecommendedActions))
	for i, a := range analysis.RecommendedActions {
		actions[i] = map[string]any{
			"action_id":   a.ActionID,
			"description": a.Description,
			"action_type": a.ActionType,
			"risk_level":  string(a.RiskLevel),
			"savings":     a.EstimatedSavingsMonthly,
		}
	}
	return Component{
		Type:       ComponentActionPlan,
		Title:      "Action Plan",
		Priority:   30,
		Visibility: VisibilityVisible,
		Data: map[string]any{
			"root_cause":         analysis.RootCauseNarrative,
			"actions":            actions,
			"estimated_savings":  analysis.EstimatedMonthlySavings,
			"affected_resources": analysis.AffectedResources,
		},
	}
}

// actionEditors builds one action_editor component per recommended action.
func actionEditors(analysis *domain.AnalysisResult) []Component {
	var comps []Component
	for i, a := range analysis.RecommendedActions {
		comps = append(comps, Component{
			Type:       ComponentActionEditor,
			Title:      a.Description,
			Priority:   35 + i,
			Visibility: VisibilityVisible,
			Data: map[string]any{
				"action_id":          a.ActionID,
				"action_type":        a.ActionType,
				"risk_level":         string(a.RiskLevel),
				"parameters":         a.Parameters,
				"rollback_procedure": a.RollbackProcedure,
				"target_resource":    a.TargetResource,
			},
		})
	}
	return comps
}

// approvalQueue builds the pending-approval component.
func approvalQueue(state domain.FinOpsState) Component {
	return Component{
		Type:       ComponentApprovalQueue,
		Title:      "Approval Required",
		Priority:   40,
		Visibility: VisibilityVisible,
		Data: map[string]any{
			"approval_status":  string(state.Approval),
			"approval_details": state.ApprovalDetails,
		},
	}
}

// executionResults builds the post-execution summary.
func executionResults(executions []domain.ExecutionResult) Component {
	results := make([]map[string]any, len(executions))
	for i, e := range executions {
		results[i] = map[string]any{
			"action_id":   e.ActionID,
			"success":     e.Success,
			"details":     e.Details,
			"executed_at": e.ExecutedAt,
			"rollback":    e.RollbackAvailable,
		}
	}
	return Component{
		Type:       ComponentExecutionResults,
		Title:      "Execution Results",
		Priority:   50,
		Visibility: VisibilityVisible,
		Data: map[string]any{
			"results": results,
		},
	}
}

// verificationDashboard builds the post-verification component.
func verificationDashboard(v *domain.VerificationResult) Component {
	return Component{
		Type:       ComponentVerificationDashboard,
		Title:      "Verification",
		Priority:   60,
		Visibility: VisibilityVisible,
		Data: map[string]any{
			"cost_reduction_observed": v.CostReductionObserved,
			"observed_savings_daily":  v.ObservedSavingsDaily,
			"service_health_ok":       v.ServiceHealthOK,
			"health_check_details":    v.HealthCheckDetails,
			"recommendation":          string(v.Recommendation),
		},
	}
}
