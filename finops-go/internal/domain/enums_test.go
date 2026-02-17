package domain

import "testing"

func TestAnomalySeverityValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		sev   AnomalySeverity
		valid bool
	}{
		{name: "low", sev: SeverityLow, valid: true},
		{name: "medium", sev: SeverityMedium, valid: true},
		{name: "high", sev: SeverityHigh, valid: true},
		{name: "critical", sev: SeverityCritical, valid: true},
		{name: "bogus", sev: AnomalySeverity("bogus"), valid: false},
		{name: "empty", sev: AnomalySeverity(""), valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.sev.Valid(); got != tt.valid {
				t.Errorf("AnomalySeverity(%q).Valid() = %v, want %v", tt.sev, got, tt.valid)
			}
		})
	}
}

func TestAnomalySeverityStringValues(t *testing.T) {
	t.Parallel()
	// Must match Python enum values exactly.
	tests := []struct {
		sev  AnomalySeverity
		want string
	}{
		{SeverityLow, "low"},
		{SeverityMedium, "medium"},
		{SeverityHigh, "high"},
		{SeverityCritical, "critical"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if string(tt.sev) != tt.want {
				t.Errorf("AnomalySeverity: got %q, want %q", tt.sev, tt.want)
			}
		})
	}
}

func TestAnomalyCategoryValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		cat   AnomalyCategory
		valid bool
	}{
		{name: "expected_growth", cat: CategoryExpectedGrowth, valid: true},
		{name: "deploy_related", cat: CategoryDeployRelated, valid: true},
		{name: "config_drift", cat: CategoryConfigDrift, valid: true},
		{name: "pricing_change", cat: CategoryPricingChange, valid: true},
		{name: "credits_refunds_fees", cat: CategoryCreditsRefundsFees, valid: true},
		{name: "marketplace", cat: CategoryMarketplace, valid: true},
		{name: "data_transfer", cat: CategoryDataTransfer, valid: true},
		{name: "k8s_cost_shift", cat: CategoryK8sCostShift, valid: true},
		{name: "commitment_coverage_drift", cat: CategoryCommitmentCoverageDrift, valid: true},
		{name: "unknown", cat: CategoryUnknown, valid: true},
		{name: "bogus", cat: AnomalyCategory("bogus"), valid: false},
		{name: "empty", cat: AnomalyCategory(""), valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cat.Valid(); got != tt.valid {
				t.Errorf("AnomalyCategory(%q).Valid() = %v, want %v", tt.cat, got, tt.valid)
			}
		})
	}
}

func TestAnomalyCategoryStringValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cat  AnomalyCategory
		want string
	}{
		{CategoryExpectedGrowth, "expected_growth"},
		{CategoryDeployRelated, "deploy_related"},
		{CategoryConfigDrift, "config_drift"},
		{CategoryPricingChange, "pricing_change"},
		{CategoryCreditsRefundsFees, "credits_refunds_fees"},
		{CategoryMarketplace, "marketplace"},
		{CategoryDataTransfer, "data_transfer"},
		{CategoryK8sCostShift, "k8s_cost_shift"},
		{CategoryCommitmentCoverageDrift, "commitment_coverage_drift"},
		{CategoryUnknown, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if string(tt.cat) != tt.want {
				t.Errorf("AnomalyCategory: got %q, want %q", tt.cat, tt.want)
			}
		})
	}
}

func TestApprovalStatusValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stat  ApprovalStatus
		valid bool
	}{
		{name: "pending", stat: ApprovalPending, valid: true},
		{name: "approved", stat: ApprovalApproved, valid: true},
		{name: "denied", stat: ApprovalDenied, valid: true},
		{name: "auto_approved", stat: ApprovalAutoApproved, valid: true},
		{name: "timed_out", stat: ApprovalTimedOut, valid: true},
		{name: "bogus", stat: ApprovalStatus("bogus"), valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.stat.Valid(); got != tt.valid {
				t.Errorf("ApprovalStatus(%q).Valid() = %v, want %v", tt.stat, got, tt.valid)
			}
		})
	}
}

func TestActionRiskLevelValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		risk  ActionRiskLevel
		valid bool
	}{
		{name: "low", risk: RiskLow, valid: true},
		{name: "low_medium", risk: RiskLowMedium, valid: true},
		{name: "medium", risk: RiskMedium, valid: true},
		{name: "high", risk: RiskHigh, valid: true},
		{name: "critical", risk: RiskCritical, valid: true},
		{name: "bogus", risk: ActionRiskLevel("bogus"), valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.risk.Valid(); got != tt.valid {
				t.Errorf("ActionRiskLevel(%q).Valid() = %v, want %v", tt.risk, got, tt.valid)
			}
		})
	}
}

func TestRiskScoreMap(t *testing.T) {
	t.Parallel()
	// Explicit scores must match Python RISK_SCORE exactly.
	tests := []struct {
		level ActionRiskLevel
		want  int
	}{
		{RiskLow, 10},
		{RiskLowMedium, 20},
		{RiskMedium, 30},
		{RiskHigh, 40},
		{RiskCritical, 50},
	}
	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			t.Parallel()
			got, ok := RiskScore[tt.level]
			if !ok {
				t.Fatalf("RiskScore missing key %q", tt.level)
			}
			if got != tt.want {
				t.Errorf("RiskScore[%q] = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestRiskScoreFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		level   ActionRiskLevel
		want    int
		wantErr bool
	}{
		{name: "low", level: RiskLow, want: 10},
		{name: "critical", level: RiskCritical, want: 50},
		{name: "unknown", level: ActionRiskLevel("bogus"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RiskScoreFor(tt.level)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RiskScoreFor(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("RiskScoreFor(%q) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestVerificationRecommendationValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		rec   VerificationRecommendation
		valid bool
	}{
		{name: "close", rec: RecommendClose, valid: true},
		{name: "rollback", rec: RecommendRollback, valid: true},
		{name: "escalate", rec: RecommendEscalate, valid: true},
		{name: "monitor", rec: RecommendMonitor, valid: true},
		{name: "bogus", rec: VerificationRecommendation("bogus"), valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.rec.Valid(); got != tt.valid {
				t.Errorf("VerificationRecommendation(%q).Valid() = %v, want %v", tt.rec, got, tt.valid)
			}
		})
	}
}
