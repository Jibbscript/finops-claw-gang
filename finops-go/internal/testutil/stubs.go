package testutil

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/finops-claw-gang/finops-go/internal/connectors/awsdoctor"
	"github.com/finops-claw-gang/finops-go/internal/domain"
)

// StubCost satisfies triage.CostFetcher, analysis.CostQuerier, and verifier.CostChecker.
type StubCost struct {
	FixturesDir string
}

func (s *StubCost) load(name string, target any) error {
	data, err := os.ReadFile(filepath.Join(s.FixturesDir, name))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (s *StubCost) GetCostTimeseries(service, accountID, startDate, endDate string) (map[string]any, error) {
	var m map[string]any
	err := s.load("cost_timeseries.json", &m)
	return m, err
}

func (s *StubCost) GetCURLineItems(accountID, startDate, endDate string, service string) ([]map[string]any, error) {
	var items []map[string]any
	err := s.load("cur_line_items.json", &items)
	return items, err
}

func (s *StubCost) GetRICoverage(accountID, startDate, endDate string) (map[string]any, error) {
	var m map[string]any
	err := s.load("ri_coverage.json", &m)
	return m, err
}

func (s *StubCost) GetRIUtilization(accountID, startDate, endDate string) (map[string]any, error) {
	var m map[string]any
	err := s.load("ri_utilization.json", &m)
	return m, err
}

func (s *StubCost) GetSPCoverage(accountID, startDate, endDate string) (map[string]any, error) {
	var m map[string]any
	err := s.load("sp_coverage.json", &m)
	return m, err
}

func (s *StubCost) GetSPUtilization(accountID, startDate, endDate string) (map[string]any, error) {
	var m map[string]any
	err := s.load("sp_utilization.json", &m)
	return m, err
}

// StubInfra satisfies triage.InfraQuerier and executor.TagFetcher.
type StubInfra struct {
	FixturesDir string
}

func (s *StubInfra) load(name string, target any) error {
	data, err := os.ReadFile(filepath.Join(s.FixturesDir, name))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (s *StubInfra) RecentDeploys(service string) ([]map[string]any, error) {
	var deploys []map[string]any
	err := s.load("deploys.json", &deploys)
	return deploys, err
}

func (s *StubInfra) CloudWatchMetrics(resourceID, metricName, namespace string) (map[string]any, error) {
	var m map[string]any
	err := s.load("cloudwatch_metrics.json", &m)
	return m, err
}

func (s *StubInfra) ResourceTags(resourceARN string) (map[string]string, error) {
	var tags map[string]string
	err := s.load("resource_tags.json", &tags)
	return tags, err
}

// StubKubeCost satisfies triage.KubeCostQuerier.
type StubKubeCost struct {
	FixturesDir string
}

func (s *StubKubeCost) Allocation(window, aggregate string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(s.FixturesDir, "kubecost_allocation.json"))
	if err != nil {
		return nil, err
	}
	var m map[string]any
	err = json.Unmarshal(data, &m)
	return m, err
}

// StubAWSDoctor satisfies triage.WasteQuerier using golden fixtures.
// It loads the real aws-doctor JSON format and delegates to awsdoctor.MapWasteFindings
// for consistent mapping behavior with production code.
type StubAWSDoctor struct {
	FixturesDir string
}

func (s *StubAWSDoctor) Waste(_ context.Context, _, region, _ string) ([]domain.WasteFinding, error) {
	data, err := os.ReadFile(filepath.Join(s.FixturesDir, "waste_report.json"))
	if err != nil {
		return nil, err
	}
	var report awsdoctor.WasteReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return awsdoctor.MapWasteFindings(report, region), nil
}

// GoldenDir returns the absolute path to the tests/golden directory.
// It walks up from the caller's file to find the repo root.
func GoldenDir() string {
	// Use runtime.Caller to find the source file location, then navigate up.
	// testutil/ is at internal/testutil/, golden is at tests/golden/
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "tests", "golden")
}
