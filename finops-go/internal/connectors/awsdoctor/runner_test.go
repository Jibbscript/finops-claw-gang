package awsdoctor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func goldenDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "tests", "golden")
}

func TestMapWasteFindings(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(goldenDir(), "waste_report.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var report WasteReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	if !report.HasWaste {
		t.Error("expected has_waste=true")
	}
	if report.AccountID != "123456789012" {
		t.Errorf("account_id = %q, want 123456789012", report.AccountID)
	}

	findings := MapWasteFindings(report, "us-east-1")

	// Expect: 1 stopped instance + 1 unattached volume + 1 stopped volume + 1 orphaned snapshot + 1 elastic IP = 5
	if len(findings) != 5 {
		t.Fatalf("expected 5 findings, got %d", len(findings))
	}

	// Check stopped instance finding
	var foundEC2 bool
	for _, f := range findings {
		if f.ResourceType == "EC2" && f.ResourceID == "i-0123456789abcdef0" {
			foundEC2 = true
			if f.Region != "us-east-1" {
				t.Errorf("EC2 finding region = %q, want us-east-1", f.Region)
			}
			if f.ResourceARN == "" {
				t.Error("EC2 finding missing ARN")
			}
		}
	}
	if !foundEC2 {
		t.Error("expected EC2 stopped instance finding")
	}

	// Check EBS volume has savings estimate
	for _, f := range findings {
		if f.ResourceType == "EBS" && f.ResourceID == "vol-0aaa111222333abcd" {
			if f.EstimatedMonthlySavings <= 0 {
				t.Error("expected positive savings for unattached EBS volume")
			}
		}
	}
}

func TestMapTrendMetrics(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(goldenDir(), "trend_report.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var report TrendReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	if len(report.Months) != 6 {
		t.Fatalf("expected 6 months, got %d", len(report.Months))
	}

	direction, velocity := MapTrendMetrics(report)
	if direction != "increasing" {
		t.Errorf("direction = %q, want increasing", direction)
	}
	if velocity <= 0 {
		t.Errorf("velocity = %f, expected positive", velocity)
	}
}

func TestMapTrendMetrics_Stable(t *testing.T) {
	t.Parallel()
	report := TrendReport{
		Months: []MonthCost{
			{Start: "2026-01-01", Total: 1000},
			{Start: "2026-02-01", Total: 1005},
		},
	}
	direction, _ := MapTrendMetrics(report)
	if direction != "stable" {
		t.Errorf("direction = %q, want stable", direction)
	}
}

func TestMapTrendMetrics_TooFewMonths(t *testing.T) {
	t.Parallel()
	report := TrendReport{Months: []MonthCost{{Start: "2026-01-01", Total: 1000}}}
	direction, velocity := MapTrendMetrics(report)
	if direction != "stable" {
		t.Errorf("direction = %q, want stable", direction)
	}
	if velocity != 0 {
		t.Errorf("velocity = %f, want 0", velocity)
	}
}

func TestBinaryRunner_BinaryNotFound(t *testing.T) {
	t.Parallel()
	runner := NewBinaryRunner("/nonexistent/aws-doctor-fake")
	_, err := runner.Waste(context.Background(), RunOpts{})
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestBinaryRunner_WasteWithMockScript(t *testing.T) {
	t.Parallel()

	fixture := filepath.Join(goldenDir(), "waste_report.json")

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "aws-doctor")
	scriptContent := "#!/bin/sh\ncat " + fixture + "\n"
	if err := os.WriteFile(script, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	runner := NewBinaryRunner(script)
	report, err := runner.Waste(context.Background(), RunOpts{Region: "us-east-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !report.HasWaste {
		t.Error("expected has_waste=true")
	}
	if report.AccountID != "123456789012" {
		t.Errorf("account_id = %q", report.AccountID)
	}
}

func TestBinaryRunner_TrendWithMockScript(t *testing.T) {
	t.Parallel()

	fixture := filepath.Join(goldenDir(), "trend_report.json")

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "aws-doctor")
	scriptContent := "#!/bin/sh\ncat " + fixture + "\n"
	if err := os.WriteFile(script, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	runner := NewBinaryRunner(script)
	report, err := runner.Trend(context.Background(), RunOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Months) != 6 {
		t.Fatalf("expected 6 months, got %d", len(report.Months))
	}
}

func TestBinaryRunner_InvalidJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "aws-doctor")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'not json'\n"), 0755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	runner := NewBinaryRunner(script)
	_, err := runner.Waste(context.Background(), RunOpts{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBinaryRunner_NonZeroExit(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "aws-doctor")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'error' >&2\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	runner := NewBinaryRunner(script)
	_, err := runner.Waste(context.Background(), RunOpts{})
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
}
