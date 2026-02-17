// Package policy implements the deterministic policy engine that decides
// whether recommended actions are auto-approved, require human approval, or
// are denied outright. It also provides an executor safety gate that prevents
// execution of critical actions or actions targeting protected resources.
package policy

import (
	"fmt"

	"github.com/finops-claw-gang/finops-go/internal/domain"
)

// PolicyDecision captures the approval outcome and a human-readable explanation.
type PolicyDecision struct {
	Approval domain.ApprovalStatus
	Details  string
}

// PolicyEngine evaluates a set of recommended actions against risk-score
// thresholds and returns an approval decision. LLMs are never in this path.
type PolicyEngine struct {
	AutoApproveMaxRisk domain.ActionRiskLevel
	DenyMinRisk        domain.ActionRiskLevel
}

// NewPolicyEngine returns an engine with the default thresholds:
// auto-approve up to low risk, deny at critical risk.
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		AutoApproveMaxRisk: domain.RiskLow,
		DenyMinRisk:        domain.RiskCritical,
	}
}

// MaxRisk returns the highest risk level present in the given actions,
// determined by the explicit RiskScore map (not enum ordering).
// The caller must ensure actions is non-empty.
func (pe *PolicyEngine) MaxRisk(actions []domain.RecommendedAction) domain.ActionRiskLevel {
	maxLevel := actions[0].RiskLevel
	maxScore := domain.RiskScore[maxLevel]
	for _, a := range actions[1:] {
		s := domain.RiskScore[a.RiskLevel]
		if s > maxScore {
			maxScore = s
			maxLevel = a.RiskLevel
		}
	}
	return maxLevel
}

// Decide evaluates the recommended actions and returns a PolicyDecision.
//
// Rules:
//  1. No actions → denied ("no recommended actions").
//  2. Max risk >= deny threshold → denied.
//  3. Max risk <= auto-approve threshold → auto-approved.
//  4. Otherwise → pending (requires human approval).
func (pe *PolicyEngine) Decide(actions []domain.RecommendedAction) PolicyDecision {
	if len(actions) == 0 {
		return PolicyDecision{
			Approval: domain.ApprovalDenied,
			Details:  "no recommended actions",
		}
	}

	maxRisk := pe.MaxRisk(actions)
	maxRiskScore := domain.RiskScore[maxRisk]

	if maxRiskScore >= domain.RiskScore[pe.DenyMinRisk] {
		return PolicyDecision{
			Approval: domain.ApprovalDenied,
			Details:  fmt.Sprintf("critical-risk action(s) present: %s; manual-only", maxRisk),
		}
	}

	if maxRiskScore <= domain.RiskScore[pe.AutoApproveMaxRisk] {
		return PolicyDecision{
			Approval: domain.ApprovalAutoApproved,
			Details:  fmt.Sprintf("auto-approved; max risk=%s", maxRisk),
		}
	}

	return PolicyDecision{
		Approval: domain.ApprovalPending,
		Details:  fmt.Sprintf("requires human approval; max risk=%s", maxRisk),
	}
}

// EnforceExecutorSafety is a hard gate invoked before any action execution.
// It returns a non-nil error if:
//   - The approval status is not approved or auto_approved.
//   - Any action has critical risk level.
//   - Any action targets a resource tagged "do-not-modify" or "manual-only".
func EnforceExecutorSafety(
	approval domain.ApprovalStatus,
	actions []domain.RecommendedAction,
	resourceTagsByARN map[string]map[string]string,
) error {
	if approval != domain.ApprovalApproved && approval != domain.ApprovalAutoApproved {
		return fmt.Errorf("cannot execute: approval status is %s", approval)
	}

	for _, a := range actions {
		if a.RiskLevel == domain.RiskCritical {
			return fmt.Errorf("refuse to execute critical action %s", a.ActionID)
		}

		if a.TargetResource != "" {
			tags, ok := resourceTagsByARN[a.TargetResource]
			if ok {
				if tags["do-not-modify"] == "true" || tags["manual-only"] == "true" {
					return fmt.Errorf("refuse to execute on tagged resource %s: %v", a.TargetResource, tags)
				}
			}
		}
	}

	return nil
}
