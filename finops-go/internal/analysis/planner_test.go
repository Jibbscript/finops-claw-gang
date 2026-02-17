package analysis

import (
	"fmt"
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
)

func TestAnalyzeAndRecommend(t *testing.T) {
	t.Parallel()
	goldenDir := testutil.GoldenDir()
	cost := &testutil.StubCost{FixturesDir: goldenDir}

	result, err := AnalyzeAndRecommend("123456789012", "EC2", "2026-02-01", "2026-02-16", cost)
	if err != nil {
		t.Fatalf("AnalyzeAndRecommend: %v", err)
	}

	if len(result.RecommendedActions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.RecommendedActions))
	}

	action := result.RecommendedActions[0]

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{name: "action_type", check: func(t *testing.T) {
			t.Helper()
			if action.ActionType != "create_budget_alert" {
				t.Errorf("action_type = %q, want %q", action.ActionType, "create_budget_alert")
			}
		}},
		{name: "risk_level", check: func(t *testing.T) {
			t.Helper()
			if action.RiskLevel != domain.RiskLow {
				t.Errorf("risk_level = %q, want %q", action.RiskLevel, domain.RiskLow)
			}
		}},
		{name: "target_resource", check: func(t *testing.T) {
			t.Helper()
			if action.TargetResource != "budget:EC2:123456789012" {
				t.Errorf("target_resource = %q, want %q", action.TargetResource, "budget:EC2:123456789012")
			}
		}},
		{name: "confidence", check: func(t *testing.T) {
			t.Helper()
			if result.Confidence != 0.4 {
				t.Errorf("confidence = %f, want 0.4", result.Confidence)
			}
		}},
		{name: "action_id non-empty", check: func(t *testing.T) {
			t.Helper()
			if action.ActionID == "" {
				t.Error("expected non-empty action_id")
			}
		}},
		{name: "schema valid", check: func(t *testing.T) {
			t.Helper()
			if err := domain.ValidateRecommendedAction(action); err != nil {
				t.Errorf("action fails validation: %v", err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t)
		})
	}
}

func TestAnalyzeWaste(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		findings         []domain.WasteFinding
		wantActions      int
		wantTotalSavings float64
		wantConfidence   float64
	}{
		{
			name:             "empty findings",
			findings:         nil,
			wantActions:      0,
			wantTotalSavings: 0,
			wantConfidence:   0.85,
		},
		{
			name: "single EC2 finding",
			findings: []domain.WasteFinding{
				{ResourceType: "EC2", ResourceID: "i-abc", ResourceARN: "arn:...:instance/i-abc", Reason: "stopped", EstimatedMonthlySavings: 0, Region: "us-east-1"},
			},
			wantActions:      1,
			wantTotalSavings: 0,
			wantConfidence:   0.85,
		},
		{
			name: "multiple resource types",
			findings: []domain.WasteFinding{
				{ResourceType: "EC2", ResourceID: "i-abc", ResourceARN: "arn:...", Reason: "stopped", EstimatedMonthlySavings: 0},
				{ResourceType: "EBS", ResourceID: "vol-abc", ResourceARN: "arn:...", Reason: "unattached", EstimatedMonthlySavings: 8.0},
				{ResourceType: "Snapshot", ResourceID: "snap-abc", ResourceARN: "arn:...", Reason: "orphaned", EstimatedMonthlySavings: 2.5},
				{ResourceType: "ElasticIP", ResourceID: "eipalloc-abc", ResourceARN: "arn:...", Reason: "unused", EstimatedMonthlySavings: 3.60},
				{ResourceType: "LoadBalancer", ResourceID: "my-lb", ResourceARN: "arn:...", Reason: "no targets", EstimatedMonthlySavings: 16.20},
				{ResourceType: "AMI", ResourceID: "ami-abc", ResourceARN: "arn:...", Reason: "unused", EstimatedMonthlySavings: 1.0},
				{ResourceType: "KeyPair", ResourceID: "kp-abc", ResourceARN: "arn:...", Reason: "unused", EstimatedMonthlySavings: 0},
			},
			wantActions:      7,
			wantTotalSavings: 31.30,
			wantConfidence:   0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := AnalyzeWaste(tt.findings)

			if len(result.RecommendedActions) != tt.wantActions {
				t.Fatalf("actions = %d, want %d", len(result.RecommendedActions), tt.wantActions)
			}
			if result.Confidence != tt.wantConfidence {
				t.Errorf("confidence = %f, want %f", result.Confidence, tt.wantConfidence)
			}
			diff := result.EstimatedMonthlySavings - tt.wantTotalSavings
			if diff < -0.01 || diff > 0.01 {
				t.Errorf("total savings = %f, want %f", result.EstimatedMonthlySavings, tt.wantTotalSavings)
			}
		})
	}
}

func TestAnalyzeWaste_ActionTemplates(t *testing.T) {
	t.Parallel()

	// Verify each known resource type maps to the correct action, risk, and rollback.
	templateTests := []struct {
		resourceType string
		wantAction   string
		wantRisk     domain.ActionRiskLevel
		wantRollback string
	}{
		{"EC2", "terminate_instance", domain.RiskLow, "launch replacement from AMI or backup"},
		{"EBS", "delete_volume", domain.RiskMedium, "restore volume from snapshot"},
		{"Snapshot", "delete_snapshot", domain.RiskMedium, "no rollback — snapshot data is permanently lost"},
		{"ElasticIP", "release_elastic_ip", domain.RiskLow, "allocate new Elastic IP and update DNS"},
		{"LoadBalancer", "delete_load_balancer", domain.RiskMedium, "recreate load balancer with same configuration"},
		{"AMI", "deregister_ami", domain.RiskLow, "re-create AMI from running instance"},
		{"KeyPair", "delete_key_pair", domain.RiskLow, "create new key pair and update instances"},
	}

	for _, tt := range templateTests {
		t.Run(tt.resourceType, func(t *testing.T) {
			t.Parallel()
			finding := domain.WasteFinding{
				ResourceType: tt.resourceType,
				ResourceID:   fmt.Sprintf("test-%s-id", tt.resourceType),
				ResourceARN:  fmt.Sprintf("arn:aws:ec2:us-east-1::test/%s", tt.resourceType),
			}
			result := AnalyzeWaste([]domain.WasteFinding{finding})
			if len(result.RecommendedActions) != 1 {
				t.Fatalf("expected 1 action, got %d", len(result.RecommendedActions))
			}
			action := result.RecommendedActions[0]
			if action.ActionType != tt.wantAction {
				t.Errorf("action_type = %q, want %q", action.ActionType, tt.wantAction)
			}
			if action.RiskLevel != tt.wantRisk {
				t.Errorf("risk_level = %q, want %q", action.RiskLevel, tt.wantRisk)
			}
			if action.RollbackProcedure != tt.wantRollback {
				t.Errorf("rollback = %q, want %q", action.RollbackProcedure, tt.wantRollback)
			}
			if action.ActionID == "" {
				t.Error("expected non-empty action_id")
			}
			if action.TargetResource != finding.ResourceARN {
				t.Errorf("target_resource = %q, want %q", action.TargetResource, finding.ResourceARN)
			}
		})
	}
}

func TestAnalyzeWaste_UnknownResourceType(t *testing.T) {
	t.Parallel()

	finding := domain.WasteFinding{
		ResourceType: "SomeFutureType",
		ResourceID:   "ft-123",
		ResourceARN:  "arn:aws:ec2:us-east-1::future/ft-123",
	}
	result := AnalyzeWaste([]domain.WasteFinding{finding})
	if len(result.RecommendedActions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.RecommendedActions))
	}
	action := result.RecommendedActions[0]
	if action.ActionType != "review_resource" {
		t.Errorf("action_type = %q, want review_resource", action.ActionType)
	}
	if action.RiskLevel != domain.RiskMedium {
		t.Errorf("risk_level = %q, want medium", action.RiskLevel)
	}
	if action.RollbackProcedure != "manual review required" {
		t.Errorf("rollback = %q, want 'manual review required'", action.RollbackProcedure)
	}
}
