package triage

import (
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
)

func TestSeverityFromDelta(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		delta float64
		want  domain.AnomalySeverity
	}{
		{name: "low at 0", delta: 0, want: domain.SeverityLow},
		{name: "low at 100", delta: 100, want: domain.SeverityLow},
		{name: "low at 199", delta: 199, want: domain.SeverityLow},
		{name: "medium at 200", delta: 200, want: domain.SeverityMedium},
		{name: "medium at 999", delta: 999, want: domain.SeverityMedium},
		{name: "high at 1000", delta: 1000, want: domain.SeverityHigh},
		{name: "high at 4999", delta: 4999, want: domain.SeverityHigh},
		{name: "critical at 5000", delta: 5000, want: domain.SeverityCritical},
		{name: "critical at 10000", delta: 10000, want: domain.SeverityCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SeverityFromDelta(tt.delta)
			if got != tt.want {
				t.Errorf("SeverityFromDelta(%v) = %q, want %q", tt.delta, got, tt.want)
			}
		})
	}
}

func TestPctChange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		newVal, oldVal float64
		want           float64
	}{
		{name: "10% increase", newVal: 110, oldVal: 100, want: 0.1},
		{name: "100% increase", newVal: 200, oldVal: 100, want: 1.0},
		{name: "50% decrease", newVal: 50, oldVal: 100, want: -0.5},
		{name: "zero oldVal nonzero newVal", newVal: 100, oldVal: 0, want: 1.0},
		{name: "both zero", newVal: 0, oldVal: 0, want: 0.0},
		{name: "no change", newVal: 100, oldVal: 100, want: 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PctChange(tt.newVal, tt.oldVal)
			diff := got - tt.want
			if diff < -0.001 || diff > 0.001 {
				t.Errorf("PctChange(%v, %v) = %v, want %v", tt.newVal, tt.oldVal, got, tt.want)
			}
		})
	}
}

func TestTriageWithFixtures(t *testing.T) {
	t.Parallel()
	goldenDir := testutil.GoldenDir()
	cost := &testutil.StubCost{FixturesDir: goldenDir}
	infra := &testutil.StubInfra{FixturesDir: goldenDir}
	kube := &testutil.StubKubeCost{FixturesDir: goldenDir}

	anomaly := domain.CostAnomaly{
		AnomalyID:         "test-001",
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

	result, err := Triage(anomaly, cost, infra, kube, "", "")
	if err != nil {
		t.Fatalf("Triage: %v", err)
	}

	// With our fixtures:
	// - RI/SP coverage_delta=0.0, so no commitment drift
	// - credits=-50 < 0.2*750=150, so no credits
	// - no marketplace
	// - data transfer=250 >= 150, so should hit data_transfer
	if result.Category != domain.CategoryDataTransfer {
		t.Errorf("expected data_transfer, got %q", result.Category)
	}
	if result.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", result.Confidence)
	}
	if result.Severity != domain.SeverityMedium {
		t.Errorf("expected medium severity for delta=750, got %q", result.Severity)
	}
}

func TestTriageNilKubecost(t *testing.T) {
	t.Parallel()
	goldenDir := testutil.GoldenDir()
	cost := &testutil.StubCost{FixturesDir: goldenDir}
	infra := &testutil.StubInfra{FixturesDir: goldenDir}

	anomaly := domain.CostAnomaly{
		AnomalyID:    "test-002",
		DetectedAt:   "2026-02-16T00:00:00Z",
		Service:      "EC2",
		AccountID:    "123456789012",
		DeltaDollars: 750.0,
		DeltaPercent: 31.25,
	}

	result, err := Triage(anomaly, cost, infra, nil, "", "")
	if err != nil {
		t.Fatalf("Triage with nil kubecost: %v", err)
	}

	// Without kubecost, should still classify (data transfer likely)
	if !result.Category.Valid() {
		t.Errorf("invalid category: %q", result.Category)
	}
}

func TestTriagePriorityBranches(t *testing.T) {
	t.Parallel()

	baseAnomaly := domain.CostAnomaly{
		DetectedAt:   "2026-02-16T00:00:00Z",
		Service:      "EC2",
		AccountID:    "123456789012",
		DeltaDollars: 1500.0,
		DeltaPercent: 40.0,
	}

	tests := []struct {
		name      string
		anomalyID string
		cost      CostFetcher
		infra     InfraQuerier
		kubecost  KubeCostQuerier
		wantCat   domain.AnomalyCategory
		wantConf  float64
	}{
		{
			name:      "priority 1: commitment coverage drift (RI)",
			anomalyID: "test-p1-ri",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.10},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems:   []map[string]any{},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			wantCat:  domain.CategoryCommitmentCoverageDrift,
			wantConf: 0.8,
		},
		{
			name:      "priority 1: commitment coverage drift (SP)",
			anomalyID: "test-p1-sp",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": -0.06},
				curItems:   []map[string]any{},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			wantCat:  domain.CategoryCommitmentCoverageDrift,
			wantConf: 0.8,
		},
		{
			name:      "priority 2: credits/refunds/fees",
			anomalyID: "test-p2",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems: []map[string]any{
					{"line_item_line_item_type": "Credit", "unblended_cost": -500.0},
				},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			wantCat:  domain.CategoryCreditsRefundsFees,
			wantConf: 0.75,
		},
		{
			name:      "priority 2: refunds",
			anomalyID: "test-p2-refund",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems: []map[string]any{
					{"line_item_line_item_type": "Refund", "unblended_cost": -400.0},
				},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			wantCat:  domain.CategoryCreditsRefundsFees,
			wantConf: 0.75,
		},
		{
			name:      "priority 3: marketplace charges",
			anomalyID: "test-p3",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems: []map[string]any{
					{"product_product_name": "AWS Marketplace: Datadog", "unblended_cost": 500.0},
				},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			wantCat:  domain.CategoryMarketplace,
			wantConf: 0.8,
		},
		{
			name:      "priority 3: marketplace via product code",
			anomalyID: "test-p3-code",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems: []map[string]any{
					{"line_item_product_code": "AWS Marketplace subscription", "unblended_cost": 500.0},
				},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			wantCat:  domain.CategoryMarketplace,
			wantConf: 0.8,
		},
		{
			name:      "priority 4: data transfer spike",
			anomalyID: "test-p4",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems: []map[string]any{
					{"line_item_usage_type": "USE1-DataTransfer-Out-Bytes", "unblended_cost": 500.0},
				},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			wantCat:  domain.CategoryDataTransfer,
			wantConf: 0.85,
		},
		{
			name:      "priority 5: k8s namespace allocation shift",
			anomalyID: "test-p5",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems:   []map[string]any{},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			kubecost: &mockKubeCostQuerier{
				allocation: map[string]any{
					"allocations": map[string]any{
						"data-pipeline": map[string]any{"delta": 500.0},
						"web-frontend":  map[string]any{"delta": 10.0},
					},
				},
			},
			wantCat:  domain.CategoryK8sCostShift,
			wantConf: 0.7,
		},
		{
			name:      "priority 5: k8s below threshold",
			anomalyID: "test-p5-low",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems:   []map[string]any{},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 1000.0, "current": 1000.0},
			},
			kubecost: &mockKubeCostQuerier{
				allocation: map[string]any{
					"allocations": map[string]any{
						"web": map[string]any{"delta": 1.0},
					},
				},
			},
			wantCat:  domain.CategoryUnknown,
			wantConf: 0.4,
		},
		{
			name:      "priority 6: deploy correlation",
			anomalyID: "test-p6",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems:   []map[string]any{},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{{"id": "deploy-42"}},
				metrics: map[string]any{"baseline": 0.0, "current": 0.0},
			},
			wantCat:  domain.CategoryDeployRelated,
			wantConf: 0.7,
		},
		{
			name:      "priority 7: expected growth",
			anomalyID: "test-p7",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems:   []map[string]any{},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 1000.0, "current": 1300.0}, // 30% increase
			},
			kubecost: nil,
			wantCat:  domain.CategoryExpectedGrowth,
			wantConf: 0.8,
		},
		{
			name:      "priority 8: unknown fallback",
			anomalyID: "test-p8",
			cost: &mockCostFetcher{
				riCoverage: map[string]any{"coverage_delta": 0.0},
				spCoverage: map[string]any{"coverage_delta": 0.0},
				curItems:   []map[string]any{},
			},
			infra: &mockInfraQuerier{
				deploys: []map[string]any{},
				metrics: map[string]any{"baseline": 1000.0, "current": 1000.0}, // 0% usage change
			},
			wantCat:  domain.CategoryUnknown,
			wantConf: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			anomaly := baseAnomaly
			anomaly.AnomalyID = tt.anomalyID

			// For expected growth test, adjust delta_percent to match usage change
			if tt.wantCat == domain.CategoryExpectedGrowth {
				anomaly.DeltaPercent = 30.0 // cost_pct = 0.30, matches usage_pct = 0.30
				anomaly.DeltaDollars = 100.0
			}

			result, err := Triage(anomaly, tt.cost, tt.infra, tt.kubecost, "", "")
			if err != nil {
				t.Fatalf("Triage: %v", err)
			}
			if result.Category != tt.wantCat {
				t.Errorf("category = %q, want %q", result.Category, tt.wantCat)
			}
			if result.Confidence != tt.wantConf {
				t.Errorf("confidence = %f, want %f", result.Confidence, tt.wantConf)
			}
			if !result.Category.Valid() {
				t.Errorf("invalid category: %q", result.Category)
			}
			if !result.Severity.Valid() {
				t.Errorf("invalid severity: %q", result.Severity)
			}
		})
	}
}

// --- mock implementations for targeted triage tests ---

type mockCostFetcher struct {
	riCoverage map[string]any
	spCoverage map[string]any
	curItems   []map[string]any
}

func (m *mockCostFetcher) GetRICoverage(_, _, _ string) (map[string]any, error) {
	return m.riCoverage, nil
}

func (m *mockCostFetcher) GetSPCoverage(_, _, _ string) (map[string]any, error) {
	return m.spCoverage, nil
}

func (m *mockCostFetcher) GetCURLineItems(_, _, _, _ string) ([]map[string]any, error) {
	return m.curItems, nil
}

type mockInfraQuerier struct {
	deploys []map[string]any
	metrics map[string]any
}

func (m *mockInfraQuerier) RecentDeploys(_ string) ([]map[string]any, error) {
	return m.deploys, nil
}

func (m *mockInfraQuerier) CloudWatchMetrics(_, _, _ string) (map[string]any, error) {
	return m.metrics, nil
}

type mockKubeCostQuerier struct {
	allocation map[string]any
}

func (m *mockKubeCostQuerier) Allocation(_, _ string) (map[string]any, error) {
	return m.allocation, nil
}
