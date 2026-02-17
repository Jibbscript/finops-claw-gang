package awsdoctor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/finops-claw-gang/finops-go/internal/domain"
)

// Runner is the interface for invoking aws-doctor.
type Runner interface {
	Waste(ctx context.Context, opts RunOpts) (WasteReport, error)
	Trend(ctx context.Context, opts RunOpts) (TrendReport, error)
}

// BinaryRunner shells out to the aws-doctor binary.
type BinaryRunner struct {
	binaryPath string
	timeout    time.Duration
}

// NewBinaryRunner creates a BinaryRunner that invokes the given binary path.
func NewBinaryRunner(binaryPath string) *BinaryRunner {
	return &BinaryRunner{
		binaryPath: binaryPath,
		timeout:    5 * time.Minute,
	}
}

// Waste runs `aws-doctor --waste --output json` and parses the result.
func (r *BinaryRunner) Waste(ctx context.Context, opts RunOpts) (WasteReport, error) {
	args := []string{"--waste", "--output", "json"}
	args = appendProfileRegion(args, opts)

	out, err := r.run(ctx, args)
	if err != nil {
		return WasteReport{}, fmt.Errorf("aws-doctor --waste: %w", err)
	}

	var report WasteReport
	if err := json.Unmarshal(out, &report); err != nil {
		return WasteReport{}, fmt.Errorf("aws-doctor --waste: parse JSON: %w", err)
	}
	return report, nil
}

// Trend runs `aws-doctor --trend --output json` and parses the result.
func (r *BinaryRunner) Trend(ctx context.Context, opts RunOpts) (TrendReport, error) {
	args := []string{"--trend", "--output", "json"}
	args = appendProfileRegion(args, opts)

	out, err := r.run(ctx, args)
	if err != nil {
		return TrendReport{}, fmt.Errorf("aws-doctor --trend: %w", err)
	}

	var report TrendReport
	if err := json.Unmarshal(out, &report); err != nil {
		return TrendReport{}, fmt.Errorf("aws-doctor --trend: parse JSON: %w", err)
	}
	return report, nil
}

func (r *BinaryRunner) run(ctx context.Context, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w (stderr: %s)", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func appendProfileRegion(args []string, opts RunOpts) []string {
	if opts.Profile != "" {
		args = append(args, "--profile", opts.Profile)
	}
	if opts.Region != "" {
		args = append(args, "--region", opts.Region)
	}
	return args
}

// MapWasteFindings converts a WasteReport into domain-level WasteFindings.
// Each distinct resource category in the report is mapped to a finding with
// a concrete resource ID (never free-form).
func MapWasteFindings(report WasteReport, region string) []domain.WasteFinding {
	var findings []domain.WasteFinding

	for _, inst := range report.StoppedInstances {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "EC2",
			ResourceID:              inst.InstanceID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", region, report.AccountID, inst.InstanceID),
			Reason:                  fmt.Sprintf("instance stopped for %d days", inst.DaysAgo),
			EstimatedMonthlySavings: 0, // aws-doctor doesn't provide per-instance cost
			Region:                  region,
		})
	}

	for _, vol := range report.UnusedEBSVolumes {
		// EBS gp3 ~$0.08/GiB/month as a rough estimate
		savings := float64(vol.SizeGiB) * 0.08
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "EBS",
			ResourceID:              vol.VolumeID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:volume/%s", region, report.AccountID, vol.VolumeID),
			Reason:                  "unattached EBS volume",
			EstimatedMonthlySavings: savings,
			Region:                  region,
		})
	}

	for _, vol := range report.StoppedVolumes {
		savings := float64(vol.SizeGiB) * 0.08
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "EBS",
			ResourceID:              vol.VolumeID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:volume/%s", region, report.AccountID, vol.VolumeID),
			Reason:                  "EBS volume attached to stopped instance",
			EstimatedMonthlySavings: savings,
			Region:                  region,
		})
	}

	for _, snap := range report.OrphanedSnapshots {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "Snapshot",
			ResourceID:              snap.SnapshotID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s::snapshot/%s", region, snap.SnapshotID),
			Reason:                  snap.Reason,
			EstimatedMonthlySavings: snap.MaxPotentialSavings,
			Region:                  region,
		})
	}

	for _, snap := range report.StaleSnapshots {
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
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s:%s:elastic-ip/%s", region, report.AccountID, eip.AllocationID),
			Reason:                  "unassociated Elastic IP",
			EstimatedMonthlySavings: 3.60, // $0.005/hr
			Region:                  region,
		})
	}

	for _, lb := range report.UnusedLoadBalancers {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "LoadBalancer",
			ResourceID:              lb.Name,
			ResourceARN:             lb.ARN,
			Reason:                  "unused load balancer (no healthy targets)",
			EstimatedMonthlySavings: 16.20, // ALB ~$0.0225/hr
			Region:                  region,
		})
	}

	for _, ami := range report.UnusedAMIs {
		findings = append(findings, domain.WasteFinding{
			ResourceType:            "AMI",
			ResourceID:              ami.ImageID,
			ResourceARN:             fmt.Sprintf("arn:aws:ec2:%s::image/%s", region, ami.ImageID),
			Reason:                  "unused AMI with associated snapshots",
			EstimatedMonthlySavings: ami.MaxPotentialSaving,
			Region:                  region,
		})
	}

	return findings
}

// MapTrendMetrics extracts trend direction and velocity from a TrendReport.
// Velocity is computed as the average month-over-month percent change.
func MapTrendMetrics(report TrendReport) (direction string, velocityPct float64) {
	if len(report.Months) < 2 {
		return "stable", 0
	}

	var totalPctChange float64
	changes := 0
	for i := 1; i < len(report.Months); i++ {
		prev := report.Months[i-1].Total
		curr := report.Months[i].Total
		if prev > 0 {
			totalPctChange += (curr - prev) / prev * 100
			changes++
		}
	}

	if changes == 0 {
		return "stable", 0
	}

	velocityPct = totalPctChange / float64(changes)

	switch {
	case velocityPct > 1:
		direction = "increasing"
	case velocityPct < -1:
		direction = "decreasing"
	default:
		direction = "stable"
	}
	return direction, velocityPct
}
