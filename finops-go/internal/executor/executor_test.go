package executor

import (
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
)

func TestExecuteActions(t *testing.T) {
	t.Parallel()
	goldenDir := testutil.GoldenDir()
	infra := &testutil.StubInfra{FixturesDir: goldenDir}

	lowAction := domain.NewRecommendedAction("tag resource", "tag", domain.RiskLow, "remove tag")
	lowAction.TargetResource = "arn:aws:ec2:us-east-1:123:instance/i-abc"
	lowTags := map[string]map[string]string{
		lowAction.TargetResource: {"env": "prod", "owner": "platform"},
	}

	protectedAction := domain.NewRecommendedAction("resize", "resize", domain.RiskMedium, "revert")
	protectedAction.TargetResource = "arn:aws:ec2:us-east-1:123:instance/i-protected"
	protectedTags := map[string]map[string]string{
		protectedAction.TargetResource: {"do-not-modify": "true"},
	}

	tests := []struct {
		name       string
		approval   domain.ApprovalStatus
		actions    []domain.RecommendedAction
		tags       map[string]map[string]string
		wantErr    bool
		wantCount  int
		wantFields func(t *testing.T, results []domain.ExecutionResult)
	}{
		{
			name:      "approved executes successfully",
			approval:  domain.ApprovalAutoApproved,
			actions:   []domain.RecommendedAction{lowAction},
			tags:      lowTags,
			wantCount: 1,
			wantFields: func(t *testing.T, results []domain.ExecutionResult) {
				t.Helper()
				if !results[0].Success {
					t.Error("expected success=true")
				}
				if results[0].ActionID != lowAction.ActionID {
					t.Errorf("action_id mismatch: %q != %q", results[0].ActionID, lowAction.ActionID)
				}
				if !results[0].RollbackAvailable {
					t.Error("expected rollback_available=true")
				}
			},
		},
		{
			name:     "pending approval blocked",
			approval: domain.ApprovalPending,
			actions:  []domain.RecommendedAction{lowAction},
			wantErr:  true,
		},
		{
			name:     "denied approval blocked",
			approval: domain.ApprovalDenied,
			actions:  []domain.RecommendedAction{lowAction},
			wantErr:  true,
		},
		{
			name:     "do-not-modify tag blocked",
			approval: domain.ApprovalApproved,
			actions:  []domain.RecommendedAction{protectedAction},
			tags:     protectedTags,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exec := NewExecutor(infra)
			results, err := exec.ExecuteActions(tt.approval, tt.actions, tt.tags)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExecuteActions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(results) != tt.wantCount {
				t.Fatalf("expected %d results, got %d", tt.wantCount, len(results))
			}
			if tt.wantFields != nil {
				tt.wantFields(t, results)
			}
		})
	}
}

func TestSnapshot(t *testing.T) {
	t.Parallel()
	goldenDir := testutil.GoldenDir()
	infra := &testutil.StubInfra{FixturesDir: goldenDir}
	exec := NewExecutor(infra)

	tests := []struct {
		name     string
		target   string
		wantTags bool
	}{
		{name: "with target resource", target: "some-arn", wantTags: true},
		{name: "without target resource", target: "", wantTags: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			action := domain.NewRecommendedAction("test", "test", domain.RiskLow, "rollback")
			action.TargetResource = tt.target

			snap, err := exec.Snapshot(action)
			if err != nil {
				t.Fatalf("Snapshot: %v", err)
			}
			_, hasTags := snap["tags"]
			if hasTags != tt.wantTags {
				t.Errorf("tags present = %v, want %v", hasTags, tt.wantTags)
			}
			if !tt.wantTags && len(snap) != 0 {
				t.Errorf("expected empty snapshot, got %v", snap)
			}
		})
	}
}
