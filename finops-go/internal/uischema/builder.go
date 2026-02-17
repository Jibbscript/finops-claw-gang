package uischema

import "github.com/finops-claw-gang/finops-go/internal/domain"

const schemaVersion = "v1"

// Build constructs a UISchema from the current workflow state.
// The schema drives what the frontend renders -- no raw JSX from the backend.
func Build(state domain.FinOpsState) UISchema {
	schema := UISchema{
		Version:    schemaVersion,
		WorkflowID: state.WorkflowID,
		Phase:      state.CurrentPhase,
	}

	// Always show anomaly summary if anomaly is present.
	if state.Anomaly != nil {
		schema.Components = append(schema.Components, anomalySummary(state))
	}

	// After triage: classification + category-specific evidence.
	if state.Triage != nil {
		schema.Components = append(schema.Components, triageCard(state.Triage))
		schema.Components = append(schema.Components, categoryEvidence(state.Triage))
	}

	// After analysis: action plan + per-action editors.
	if state.Analysis != nil {
		schema.Components = append(schema.Components, actionPlan(state.Analysis))
		schema.Components = append(schema.Components, actionEditors(state.Analysis)...)
	}

	// At hil_gate with pending approval: approval queue + approve/deny actions.
	if state.Approval == domain.ApprovalPending && state.Analysis != nil {
		schema.Components = append(schema.Components, approvalQueue(state))
		schema.Actions = append(schema.Actions,
			Action{
				Type:  ActionApprove,
				Label: "Approve Actions",
			},
			Action{
				Type:  ActionDeny,
				Label: "Deny Actions",
			},
		)
		// High-risk actions need confirmation.
		if hasHighRisk(state.Analysis.RecommendedActions) {
			schema.Actions[0].Confirm = &ConfirmConfig{
				Required:        true,
				AcknowledgeText: "I understand these actions include high-risk changes",
			}
		}
	}

	// After execution: results.
	if len(state.Executions) > 0 {
		schema.Components = append(schema.Components, executionResults(state.Executions))
	}

	// After verification: dashboard + conditional rollback.
	if state.Verification != nil {
		schema.Components = append(schema.Components, verificationDashboard(state.Verification))
		if state.Verification.Recommendation == domain.RecommendRollback {
			schema.Actions = append(schema.Actions, Action{
				Type:  ActionRollback,
				Label: "Rollback Changes",
				Confirm: &ConfirmConfig{
					Required:        true,
					AcknowledgeText: "I want to rollback the executed changes",
				},
			})
		}
		if state.Verification.Recommendation == domain.RecommendEscalate {
			schema.Actions = append(schema.Actions, Action{
				Type:  ActionEscalate,
				Label: "Escalate to Engineering",
			})
		}
	}

	return schema
}

func hasHighRisk(actions []domain.RecommendedAction) bool {
	for _, a := range actions {
		if a.RiskLevel == domain.RiskHigh || a.RiskLevel == domain.RiskCritical {
			return true
		}
	}
	return false
}
