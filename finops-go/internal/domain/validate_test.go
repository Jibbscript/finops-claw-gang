package domain

import "testing"

func TestValidateCostAnomaly(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		anomaly CostAnomaly
		wantErr bool
	}{
		{
			name:    "valid",
			anomaly: CostAnomaly{AnomalyID: "abc", Service: "EC2", AccountID: "123"},
			wantErr: false,
		},
		{
			name:    "missing anomaly_id",
			anomaly: CostAnomaly{Service: "EC2", AccountID: "123"},
			wantErr: true,
		},
		{
			name:    "missing service",
			anomaly: CostAnomaly{AnomalyID: "abc", AccountID: "123"},
			wantErr: true,
		},
		{
			name:    "missing account_id",
			anomaly: CostAnomaly{AnomalyID: "abc", Service: "EC2"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCostAnomaly(tt.anomaly)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCostAnomaly() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecommendedAction(t *testing.T) {
	t.Parallel()
	valid := RecommendedAction{
		ActionID:          "abc",
		Description:       "tag resource",
		ActionType:        "tag",
		RiskLevel:         RiskLow,
		RollbackProcedure: "remove tag",
	}
	tests := []struct {
		name    string
		modify  func(RecommendedAction) RecommendedAction
		wantErr bool
	}{
		{name: "valid", modify: func(a RecommendedAction) RecommendedAction { return a }, wantErr: false},
		{name: "missing description", modify: func(a RecommendedAction) RecommendedAction { a.Description = ""; return a }, wantErr: true},
		{name: "missing action_type", modify: func(a RecommendedAction) RecommendedAction { a.ActionType = ""; return a }, wantErr: true},
		{name: "missing action_id", modify: func(a RecommendedAction) RecommendedAction { a.ActionID = ""; return a }, wantErr: true},
		{name: "missing rollback_procedure", modify: func(a RecommendedAction) RecommendedAction { a.RollbackProcedure = ""; return a }, wantErr: true},
		{name: "invalid risk_level", modify: func(a RecommendedAction) RecommendedAction { a.RiskLevel = ActionRiskLevel("bogus"); return a }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateRecommendedAction(tt.modify(valid))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecommendedAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTriageResult(t *testing.T) {
	t.Parallel()
	valid := TriageResult{
		Category:   CategoryUnknown,
		Severity:   SeverityMedium,
		Confidence: 0.5,
	}
	tests := []struct {
		name    string
		modify  func(TriageResult) TriageResult
		wantErr bool
	}{
		{name: "valid", modify: func(r TriageResult) TriageResult { return r }, wantErr: false},
		{name: "invalid category", modify: func(r TriageResult) TriageResult { r.Category = AnomalyCategory("bogus"); return r }, wantErr: true},
		{name: "invalid severity", modify: func(r TriageResult) TriageResult { r.Severity = AnomalySeverity("bogus"); return r }, wantErr: true},
		{name: "confidence > 1", modify: func(r TriageResult) TriageResult { r.Confidence = 1.5; return r }, wantErr: true},
		{name: "confidence < 0", modify: func(r TriageResult) TriageResult { r.Confidence = -0.1; return r }, wantErr: true},
		{name: "confidence = 0", modify: func(r TriageResult) TriageResult { r.Confidence = 0.0; return r }, wantErr: false},
		{name: "confidence = 1", modify: func(r TriageResult) TriageResult { r.Confidence = 1.0; return r }, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTriageResult(tt.modify(valid))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTriageResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTenantContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ctx     TenantContext
		wantErr bool
	}{
		{name: "valid", ctx: TenantContext{TenantID: "t-001"}, wantErr: false},
		{name: "empty tenant_id", ctx: TenantContext{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTenantContext(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTenantContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateVerificationResult(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		result  VerificationResult
		wantErr bool
	}{
		{name: "valid close", result: VerificationResult{Recommendation: RecommendClose}, wantErr: false},
		{name: "valid monitor", result: VerificationResult{Recommendation: RecommendMonitor}, wantErr: false},
		{name: "invalid", result: VerificationResult{Recommendation: VerificationRecommendation("bogus")}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateVerificationResult(tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVerificationResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFinOpsState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		modify  func(FinOpsState) FinOpsState
		wantErr bool
	}{
		{
			name:    "valid",
			modify:  func(s FinOpsState) FinOpsState { return s },
			wantErr: false,
		},
		{
			name:    "missing workflow_id",
			modify:  func(s FinOpsState) FinOpsState { s.WorkflowID = ""; return s },
			wantErr: true,
		},
		{
			name:    "invalid approval",
			modify:  func(s FinOpsState) FinOpsState { s.Approval = ApprovalStatus("bogus"); return s },
			wantErr: true,
		},
		{
			name:    "invalid tenant",
			modify:  func(s FinOpsState) FinOpsState { s.Tenant.TenantID = ""; return s },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewFinOpsState(NewTenantContext("t-001"))
			err := ValidateFinOpsState(tt.modify(s))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFinOpsState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
