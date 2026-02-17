package domain

import (
	"encoding/json"
	"testing"
)

func TestNewCostAnomalyDefaults(t *testing.T) {
	t.Parallel()
	a := NewCostAnomaly()
	if a.AnomalyID == "" {
		t.Error("expected non-empty AnomalyID")
	}
	if a.DetectedAt == "" {
		t.Error("expected non-empty DetectedAt")
	}
	if a.LookbackDays != 30 {
		t.Errorf("expected LookbackDays=30, got %d", a.LookbackDays)
	}
}

func TestCostAnomalyJSONRoundTrip(t *testing.T) {
	t.Parallel()
	a := CostAnomaly{
		AnomalyID:         "abc12345",
		DetectedAt:        "2026-02-16T00:00:00Z",
		Service:           "EC2",
		AccountID:         "123456789012",
		Region:            "us-east-1",
		Team:              "platform",
		ExpectedDailyCost: 2400.0,
		ActualDailyCost:   3150.0,
		DeltaDollars:      750.0,
		DeltaPercent:      31.25,
		ZScore:            3.2,
		LookbackDays:      30,
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var b CostAnomaly
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if a != b {
		t.Errorf("round-trip mismatch:\n  got  %+v\n  want %+v", b, a)
	}
}

func TestTriageEvidenceOptionalFields(t *testing.T) {
	t.Parallel()
	ev := TriageEvidence{}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// nil *float64 fields should marshal as null
	nullFields := []string{
		"ri_coverage_delta", "sp_coverage_delta", "credits_delta",
		"refunds_delta", "fees_delta", "marketplace_delta", "data_transfer_delta",
	}
	for _, field := range nullFields {
		if m[field] != nil {
			t.Errorf("expected %s to be null, got %v", field, m[field])
		}
	}
}

func TestRecommendedActionJSONRoundTrip(t *testing.T) {
	t.Parallel()
	a := NewRecommendedAction("tag resource", "tag", RiskLow, "remove tag")
	a.EstimatedSavingsMonthly = 100.0
	a.TargetResource = "arn:aws:ec2:us-east-1:123456789012:instance/i-abc"
	a.Parameters = map[string]any{"key": "team", "value": "platform"}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var b RecommendedAction
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if b.ActionID != a.ActionID {
		t.Errorf("ActionID: got %q, want %q", b.ActionID, a.ActionID)
	}
	if b.Description != a.Description {
		t.Errorf("Description: got %q, want %q", b.Description, a.Description)
	}
	if b.ActionType != a.ActionType {
		t.Errorf("ActionType: got %q, want %q", b.ActionType, a.ActionType)
	}
	if b.RiskLevel != a.RiskLevel {
		t.Errorf("RiskLevel: got %q, want %q", b.RiskLevel, a.RiskLevel)
	}
	if b.EstimatedSavingsMonthly != a.EstimatedSavingsMonthly {
		t.Errorf("EstimatedSavingsMonthly: got %f, want %f", b.EstimatedSavingsMonthly, a.EstimatedSavingsMonthly)
	}
	if b.TargetResource != a.TargetResource {
		t.Errorf("TargetResource: got %q, want %q", b.TargetResource, a.TargetResource)
	}
	if b.RollbackProcedure != a.RollbackProcedure {
		t.Errorf("RollbackProcedure: got %q, want %q", b.RollbackProcedure, a.RollbackProcedure)
	}
	if len(b.Parameters) != len(a.Parameters) {
		t.Errorf("Parameters length: got %d, want %d", len(b.Parameters), len(a.Parameters))
	}
}

func TestNewFinOpsStateDefaults(t *testing.T) {
	t.Parallel()
	tenant := NewTenantContext("t-001")
	s := NewFinOpsState(tenant)
	if s.WorkflowID == "" {
		t.Error("expected non-empty WorkflowID")
	}
	if s.CurrentPhase != "watcher" {
		t.Errorf("expected CurrentPhase=watcher, got %q", s.CurrentPhase)
	}
	if s.Approval != ApprovalPending {
		t.Errorf("expected Approval=pending, got %q", s.Approval)
	}
	if s.Tenant.DefaultRegion != "us-east-1" {
		t.Errorf("expected DefaultRegion=us-east-1, got %q", s.Tenant.DefaultRegion)
	}
}

func TestNewUUIDFormat(t *testing.T) {
	t.Parallel()
	id := newUUID()
	if len(id) != 36 {
		t.Errorf("expected UUID length 36, got %d: %q", len(id), id)
	}
	// Check hyphens at correct positions
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Errorf("UUID format incorrect: %q", id)
	}
}
