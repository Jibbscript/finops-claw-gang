package domain

import "fmt"

// ValidateCostAnomaly checks required fields on a CostAnomaly.
func ValidateCostAnomaly(a CostAnomaly) error {
	if a.AnomalyID == "" {
		return fmt.Errorf("anomaly_id is required")
	}
	if a.Service == "" {
		return fmt.Errorf("service is required")
	}
	if a.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	return nil
}

// ValidateRecommendedAction checks required fields on a RecommendedAction.
func ValidateRecommendedAction(a RecommendedAction) error {
	if a.ActionID == "" {
		return fmt.Errorf("action_id is required")
	}
	if a.Description == "" {
		return fmt.Errorf("description is required")
	}
	if a.ActionType == "" {
		return fmt.Errorf("action_type is required")
	}
	if !a.RiskLevel.Valid() {
		return fmt.Errorf("invalid risk_level: %q", a.RiskLevel)
	}
	if a.RollbackProcedure == "" {
		return fmt.Errorf("rollback_procedure is required")
	}
	return nil
}

// ValidateTriageResult checks required fields on a TriageResult.
func ValidateTriageResult(t TriageResult) error {
	if !t.Category.Valid() {
		return fmt.Errorf("invalid category: %q", t.Category)
	}
	if !t.Severity.Valid() {
		return fmt.Errorf("invalid severity: %q", t.Severity)
	}
	if t.Confidence < 0 || t.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", t.Confidence)
	}
	return nil
}

// ValidateTenantContext checks required fields on a TenantContext.
func ValidateTenantContext(t TenantContext) error {
	if t.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	return nil
}

// ValidateVerificationResult checks a VerificationResult.
func ValidateVerificationResult(v VerificationResult) error {
	if !v.Recommendation.Valid() {
		return fmt.Errorf("invalid recommendation: %q", v.Recommendation)
	}
	return nil
}

// ValidateFinOpsState checks required fields on a FinOpsState.
func ValidateFinOpsState(s FinOpsState) error {
	if s.WorkflowID == "" {
		return fmt.Errorf("workflow_id is required")
	}
	if err := ValidateTenantContext(s.Tenant); err != nil {
		return fmt.Errorf("tenant: %w", err)
	}
	if !s.Approval.Valid() {
		return fmt.Errorf("invalid approval status: %q", s.Approval)
	}
	return nil
}
