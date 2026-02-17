package policy

import (
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/domain"
)

func makeAction(risk domain.ActionRiskLevel) domain.RecommendedAction {
	return domain.NewRecommendedAction("test action", "test", risk, "rollback")
}

func TestDecide(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		actions []domain.RecommendedAction
		want    domain.ApprovalStatus
		details string
	}{
		{
			name:    "empty actions denied",
			actions: nil,
			want:    domain.ApprovalDenied,
			details: "no recommended actions",
		},
		{
			name:    "low risk auto-approved",
			actions: []domain.RecommendedAction{makeAction(domain.RiskLow)},
			want:    domain.ApprovalAutoApproved,
		},
		{
			name:    "critical risk denied",
			actions: []domain.RecommendedAction{makeAction(domain.RiskCritical)},
			want:    domain.ApprovalDenied,
		},
		{
			name:    "medium risk pending",
			actions: []domain.RecommendedAction{makeAction(domain.RiskMedium)},
			want:    domain.ApprovalPending,
		},
		{
			name:    "high risk pending",
			actions: []domain.RecommendedAction{makeAction(domain.RiskHigh)},
			want:    domain.ApprovalPending,
		},
		{
			name:    "low_medium risk pending",
			actions: []domain.RecommendedAction{makeAction(domain.RiskLowMedium)},
			want:    domain.ApprovalPending,
		},
		{
			name: "mixed low+high pending",
			actions: []domain.RecommendedAction{
				makeAction(domain.RiskLow),
				makeAction(domain.RiskHigh),
			},
			want: domain.ApprovalPending,
		},
		{
			name: "mixed low+critical denied",
			actions: []domain.RecommendedAction{
				makeAction(domain.RiskLow),
				makeAction(domain.RiskCritical),
			},
			want: domain.ApprovalDenied,
		},
		{
			name: "multiple low auto-approved",
			actions: []domain.RecommendedAction{
				makeAction(domain.RiskLow),
				makeAction(domain.RiskLow),
			},
			want: domain.ApprovalAutoApproved,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pe := NewPolicyEngine()
			d := pe.Decide(tt.actions)
			if d.Approval != tt.want {
				t.Errorf("Decide() approval = %q, want %q", d.Approval, tt.want)
			}
			if tt.details != "" && d.Details != tt.details {
				t.Errorf("Decide() details = %q, want %q", d.Details, tt.details)
			}
		})
	}
}

func TestMaxRisk(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		actions []domain.RecommendedAction
		want    domain.ActionRiskLevel
	}{
		{
			name:    "single low",
			actions: []domain.RecommendedAction{makeAction(domain.RiskLow)},
			want:    domain.RiskLow,
		},
		{
			name: "low+high+medium",
			actions: []domain.RecommendedAction{
				makeAction(domain.RiskLow),
				makeAction(domain.RiskHigh),
				makeAction(domain.RiskMedium),
			},
			want: domain.RiskHigh,
		},
		{
			name: "all critical",
			actions: []domain.RecommendedAction{
				makeAction(domain.RiskCritical),
				makeAction(domain.RiskCritical),
			},
			want: domain.RiskCritical,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pe := NewPolicyEngine()
			got := pe.MaxRisk(tt.actions)
			if got != tt.want {
				t.Errorf("MaxRisk() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnforceExecutorSafety(t *testing.T) {
	t.Parallel()
	lowAction := makeAction(domain.RiskLow)
	criticalAction := makeAction(domain.RiskCritical)

	taggedAction := makeAction(domain.RiskLow)
	taggedAction.TargetResource = "arn:aws:ec2:us-east-1:123:instance/i-abc"

	tests := []struct {
		name     string
		approval domain.ApprovalStatus
		actions  []domain.RecommendedAction
		tags     map[string]map[string]string
		wantErr  bool
	}{
		{
			name:     "approved passes",
			approval: domain.ApprovalApproved,
			actions:  []domain.RecommendedAction{lowAction},
			wantErr:  false,
		},
		{
			name:     "auto_approved passes",
			approval: domain.ApprovalAutoApproved,
			actions:  []domain.RecommendedAction{lowAction},
			wantErr:  false,
		},
		{
			name:     "pending blocked",
			approval: domain.ApprovalPending,
			actions:  []domain.RecommendedAction{lowAction},
			wantErr:  true,
		},
		{
			name:     "denied blocked",
			approval: domain.ApprovalDenied,
			actions:  []domain.RecommendedAction{lowAction},
			wantErr:  true,
		},
		{
			name:     "critical action blocked",
			approval: domain.ApprovalApproved,
			actions:  []domain.RecommendedAction{criticalAction},
			wantErr:  true,
		},
		{
			name:     "do-not-modify tag blocked",
			approval: domain.ApprovalApproved,
			actions:  []domain.RecommendedAction{taggedAction},
			tags:     map[string]map[string]string{taggedAction.TargetResource: {"do-not-modify": "true"}},
			wantErr:  true,
		},
		{
			name:     "manual-only tag blocked",
			approval: domain.ApprovalApproved,
			actions:  []domain.RecommendedAction{taggedAction},
			tags:     map[string]map[string]string{taggedAction.TargetResource: {"manual-only": "true"}},
			wantErr:  true,
		},
		{
			name:     "tagged resource without blocking tags passes",
			approval: domain.ApprovalApproved,
			actions:  []domain.RecommendedAction{taggedAction},
			tags:     map[string]map[string]string{taggedAction.TargetResource: {"env": "prod"}},
			wantErr:  false,
		},
		{
			name:     "nil tags passes",
			approval: domain.ApprovalApproved,
			actions:  []domain.RecommendedAction{taggedAction},
			tags:     nil,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := EnforceExecutorSafety(tt.approval, tt.actions, tt.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnforceExecutorSafety() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
