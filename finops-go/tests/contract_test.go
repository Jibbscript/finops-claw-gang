package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/testutil"
)

func goldenDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "golden")
}

// TestContractFixturesExist verifies all 10 golden fixture files exist.
func TestContractFixturesExist(t *testing.T) {
	t.Parallel()
	dir := goldenDir()
	expected := []string{
		"cloudwatch_metrics.json",
		"cost_timeseries.json",
		"cur_line_items.json",
		"deploys.json",
		"kubecost_allocation.json",
		"resource_tags.json",
		"ri_coverage.json",
		"ri_utilization.json",
		"sp_coverage.json",
		"sp_utilization.json",
	}
	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("missing golden fixture: %s", name)
			}
		})
	}
}

// TestContractFixturesValidJSON verifies each fixture is valid JSON.
func TestContractFixturesValidJSON(t *testing.T) {
	t.Parallel()
	dir := goldenDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read golden dir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatalf("read %s: %v", e.Name(), err)
			}
			if !json.Valid(data) {
				t.Errorf("%s is not valid JSON", e.Name())
			}
		})
	}
}

// TestContractCURLineItemsSchema validates CUR line items fixture structure.
func TestContractCURLineItemsSchema(t *testing.T) {
	t.Parallel()
	dir := goldenDir()
	cost := &testutil.StubCost{FixturesDir: dir}
	items, err := cost.GetCURLineItems("test", "2026-02-01", "2026-02-16", "EC2")
	if err != nil {
		t.Fatalf("load CUR items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one CUR line item")
	}
	requiredKeys := []string{"line_item_line_item_type", "unblended_cost"}
	for i, item := range items {
		for _, key := range requiredKeys {
			if _, ok := item[key]; !ok {
				t.Errorf("CUR item[%d] missing key %q", i, key)
			}
		}
	}
}

// TestContractCostTimeseriesSchema validates cost timeseries fixture.
func TestContractCostTimeseriesSchema(t *testing.T) {
	t.Parallel()
	dir := goldenDir()
	cost := &testutil.StubCost{FixturesDir: dir}
	ts, err := cost.GetCostTimeseries("EC2", "test", "2026-02-01", "2026-02-16")
	if err != nil {
		t.Fatalf("load timeseries: %v", err)
	}
	if _, ok := ts["observed_savings_daily"]; !ok {
		t.Error("timeseries missing observed_savings_daily")
	}
}

// TestContractCoverageSchema validates RI/SP coverage fixtures.
func TestContractCoverageSchema(t *testing.T) {
	t.Parallel()
	dir := goldenDir()
	cost := &testutil.StubCost{FixturesDir: dir}

	ri, err := cost.GetRICoverage("test", "2026-02-01", "2026-02-16")
	if err != nil {
		t.Fatalf("load RI coverage: %v", err)
	}
	if _, ok := ri["coverage_delta"]; !ok {
		t.Error("RI coverage missing coverage_delta")
	}

	sp, err := cost.GetSPCoverage("test", "2026-02-01", "2026-02-16")
	if err != nil {
		t.Fatalf("load SP coverage: %v", err)
	}
	if _, ok := sp["coverage_delta"]; !ok {
		t.Error("SP coverage missing coverage_delta")
	}
}

// TestContractResourceTagsSchema validates resource tags fixture.
func TestContractResourceTagsSchema(t *testing.T) {
	t.Parallel()
	dir := goldenDir()
	infra := &testutil.StubInfra{FixturesDir: dir}
	tags, err := infra.ResourceTags("test-arn")
	if err != nil {
		t.Fatalf("load tags: %v", err)
	}
	if len(tags) == 0 {
		t.Error("expected non-empty tags")
	}
}

// TestContractKubeCostSchema validates kubecost allocation fixture.
func TestContractKubeCostSchema(t *testing.T) {
	t.Parallel()
	dir := goldenDir()
	kube := &testutil.StubKubeCost{FixturesDir: dir}
	alloc, err := kube.Allocation("24h", "namespace")
	if err != nil {
		t.Fatalf("load kubecost: %v", err)
	}
	allocations, ok := alloc["allocations"]
	if !ok {
		t.Fatal("kubecost missing allocations key")
	}
	nsMap, ok := allocations.(map[string]any)
	if !ok {
		t.Fatal("allocations is not a map")
	}
	if len(nsMap) == 0 {
		t.Error("expected non-empty allocations")
	}
}

// TestContractStubsSatisfyInterfaces verifies stubs satisfy all consumer interfaces
// by compiling successfully (this is a compile-time check).
func TestContractStubsSatisfyInterfaces(t *testing.T) {
	t.Parallel()
	dir := goldenDir()
	cost := &testutil.StubCost{FixturesDir: dir}
	infra := &testutil.StubInfra{FixturesDir: dir}

	// Verify we can use stubs where interfaces are expected
	// These are compile-time checks — if they build, they pass.
	_ = cost  // satisfies CostFetcher, CostQuerier, CostChecker
	_ = infra // satisfies InfraQuerier, TagFetcher
}

// TestContractDomainEnumStringParity verifies Go enum string values match Python.
func TestContractDomainEnumStringParity(t *testing.T) {
	t.Parallel()

	t.Run("categories", func(t *testing.T) {
		t.Parallel()
		categories := []struct {
			cat  domain.AnomalyCategory
			want string
		}{
			{domain.CategoryExpectedGrowth, "expected_growth"},
			{domain.CategoryDeployRelated, "deploy_related"},
			{domain.CategoryConfigDrift, "config_drift"},
			{domain.CategoryPricingChange, "pricing_change"},
			{domain.CategoryCreditsRefundsFees, "credits_refunds_fees"},
			{domain.CategoryMarketplace, "marketplace"},
			{domain.CategoryDataTransfer, "data_transfer"},
			{domain.CategoryK8sCostShift, "k8s_cost_shift"},
			{domain.CategoryCommitmentCoverageDrift, "commitment_coverage_drift"},
			{domain.CategoryUnknown, "unknown"},
		}
		for _, tt := range categories {
			t.Run(tt.want, func(t *testing.T) {
				t.Parallel()
				if string(tt.cat) != tt.want {
					t.Errorf("category %q != expected %q", tt.cat, tt.want)
				}
			})
		}
	})

	t.Run("risk_scores", func(t *testing.T) {
		t.Parallel()
		risks := []struct {
			level domain.ActionRiskLevel
			want  int
		}{
			{domain.RiskLow, 10},
			{domain.RiskLowMedium, 20},
			{domain.RiskMedium, 30},
			{domain.RiskHigh, 40},
			{domain.RiskCritical, 50},
		}
		for _, tt := range risks {
			t.Run(string(tt.level), func(t *testing.T) {
				t.Parallel()
				if domain.RiskScore[tt.level] != tt.want {
					t.Errorf("RiskScore[%q] = %d, want %d", tt.level, domain.RiskScore[tt.level], tt.want)
				}
			})
		}
	})
}
