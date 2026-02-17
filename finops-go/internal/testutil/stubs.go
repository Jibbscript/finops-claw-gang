package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
// It loads the real aws-doctor JSON format and maps findings to domain types.
type StubAWSDoctor struct {
	FixturesDir string
}

// wasteReport mirrors the aws-doctor JSON shape (just the fields we need).
type wasteReport struct {
	AccountID        string           `json:"account_id"`
	HasWaste         bool             `json:"has_waste"`
	StoppedInstances []stoppedInst    `json:"stopped_instances"`
	UnusedEBSVolumes []ebsVolume      `json:"unused_ebs_volumes"`
	StoppedVolumes   []ebsVolume      `json:"stopped_instance_volumes"`
	OrphanedSnaps    []snapshotEntry  `json:"orphaned_snapshots"`
	UnusedElasticIPs []elasticIPEntry `json:"unused_elastic_ips"`
}
type stoppedInst struct {
	InstanceID string `json:"instance_id"`
	DaysAgo    int    `json:"days_ago"`
}
type ebsVolume struct {
	VolumeID string `json:"volume_id"`
	SizeGiB  int32  `json:"size_gib"`
}
type snapshotEntry struct {
	SnapshotID          string  `json:"snapshot_id"`
	Reason              string  `json:"reason"`
	MaxPotentialSavings float64 `json:"max_potential_savings"`
}
type elasticIPEntry struct {
	AllocationID string `json:"allocation_id"`
}

func (s *StubAWSDoctor) Waste(_ context.Context, accountID, region string) ([]domain.WasteFinding, error) {
	data, err := os.ReadFile(filepath.Join(s.FixturesDir, "waste_report.json"))
	if err != nil {
		return nil, err
	}
	var report wasteReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	var findings []domain.WasteFinding

	for _, inst := range report.StoppedInstances {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "EC2",
			ResourceID:              inst.InstanceID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", region, accountID, inst.InstanceID),
			Reason:                  fmt.Sprintf("instance stopped for %d days", inst.DaysAgo),
			EstimatedMonthlySavings: 0,
			Region:                  region,
		})
	}
	for _, vol := range report.UnusedEBSVolumes {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "EBS",
			ResourceID:              vol.VolumeID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:volume/%s", region, accountID, vol.VolumeID),
			Reason:                  "unattached EBS volume",
			EstimatedMonthlySavings: float64(vol.SizeGiB) * 0.08,
			Region:                  region,
		})
	}
	for _, vol := range report.StoppedVolumes {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "EBS",
			ResourceID:              vol.VolumeID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:volume/%s", region, accountID, vol.VolumeID),
			Reason:                  "EBS volume attached to stopped instance",
			EstimatedMonthlySavings: float64(vol.SizeGiB) * 0.08,
			Region:                  region,
		})
	}
	for _, snap := range report.OrphanedSnaps {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "Snapshot",
			ResourceID:              snap.SnapshotID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s::snapshot/%s", region, snap.SnapshotID),
			Reason:                  snap.Reason,
			EstimatedMonthlySavings: snap.MaxPotentialSavings,
			Region:                  region,
		})
	}
	for _, eip := range report.UnusedElasticIPs {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "ElasticIP",
			ResourceID:              eip.AllocationID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:elastic-ip/%s", region, accountID, eip.AllocationID),
			Reason:                  "unassociated Elastic IP",
			EstimatedMonthlySavings: 3.60,
			Region:                  region,
		})
	}

	return findings, nil
}

// GoldenDir returns the absolute path to the tests/golden directory.
// It walks up from the caller's file to find the repo root.
func GoldenDir() string {
	// Use runtime.Caller to find the source file location, then navigate up.
	// testutil/ is at internal/testutil/, golden is at tests/golden/
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "tests", "golden")
}
