//go:build integration

// Package tests contains integration tests that require real AWS credentials.
// Run with: go test -tags=integration ./tests -v
package tests

import (
	"context"
	"os"
	"testing"
	"time"

	awsauth "github.com/finops-claw-gang/finops-go/internal/connectors/aws"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/cloudwatch"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/codedeploy"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/costexplorer"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/tagging"
	"github.com/stretchr/testify/require"
)

func awsConfig(t *testing.T) {
	t.Helper()
	if os.Getenv("AWS_REGION") == "" {
		t.Skip("AWS_REGION not set, skipping integration test")
	}
}

func TestIntegration_CostExplorer_GetCostTimeseries(t *testing.T) {
	awsConfig(t)
	accountID := os.Getenv("AWS_ACCOUNT_ID")
	if accountID == "" {
		t.Skip("AWS_ACCOUNT_ID not set")
	}

	cfg, err := awsauth.NewAWSConfig(context.Background(), os.Getenv("AWS_REGION"), os.Getenv("AWS_PROFILE"), "")
	require.NoError(t, err)

	client := costexplorer.New(cfg)
	end := time.Now().UTC().Format("2006-01-02")
	start := time.Now().UTC().Add(-7 * 24 * time.Hour).Format("2006-01-02")

	result, err := client.GetCostTimeseries("Amazon Elastic Compute Cloud - Compute", accountID, start, end)
	require.NoError(t, err)

	_, ok := result["observed_savings_daily"]
	require.True(t, ok, "result must contain observed_savings_daily")
	_, ok = result["points"]
	require.True(t, ok, "result must contain points")
}

func TestIntegration_CostExplorer_GetRICoverage(t *testing.T) {
	awsConfig(t)
	accountID := os.Getenv("AWS_ACCOUNT_ID")
	if accountID == "" {
		t.Skip("AWS_ACCOUNT_ID not set")
	}

	cfg, err := awsauth.NewAWSConfig(context.Background(), os.Getenv("AWS_REGION"), os.Getenv("AWS_PROFILE"), "")
	require.NoError(t, err)

	client := costexplorer.New(cfg)
	end := time.Now().UTC().Format("2006-01-02")
	start := time.Now().UTC().Add(-7 * 24 * time.Hour).Format("2006-01-02")

	result, err := client.GetRICoverage(accountID, start, end)
	require.NoError(t, err)
	_, ok := result["coverage_delta"]
	require.True(t, ok, "result must contain coverage_delta")
}

func TestIntegration_CloudWatch(t *testing.T) {
	awsConfig(t)
	instanceID := os.Getenv("TEST_INSTANCE_ID")
	if instanceID == "" {
		t.Skip("TEST_INSTANCE_ID not set")
	}

	cfg, err := awsauth.NewAWSConfig(context.Background(), os.Getenv("AWS_REGION"), os.Getenv("AWS_PROFILE"), "")
	require.NoError(t, err)

	client := cloudwatch.New(cfg)
	result, err := client.CloudWatchMetrics(instanceID, "CPUUtilization", "AWS/EC2")
	require.NoError(t, err)

	_, ok := result["baseline"]
	require.True(t, ok, "result must contain baseline")
	_, ok = result["current"]
	require.True(t, ok, "result must contain current")
}

func TestIntegration_CodeDeploy(t *testing.T) {
	awsConfig(t)
	appName := os.Getenv("TEST_CODEDEPLOY_APP")
	if appName == "" {
		t.Skip("TEST_CODEDEPLOY_APP not set")
	}

	cfg, err := awsauth.NewAWSConfig(context.Background(), os.Getenv("AWS_REGION"), os.Getenv("AWS_PROFILE"), "")
	require.NoError(t, err)

	client := codedeploy.New(cfg)
	deploys, err := client.RecentDeploys(appName)
	require.NoError(t, err)
	// May be empty if no recent deploys; just verify the call succeeds.
	t.Logf("found %d recent deploys", len(deploys))
}

func TestIntegration_Tagging(t *testing.T) {
	awsConfig(t)
	resourceARN := os.Getenv("TEST_RESOURCE_ARN")
	if resourceARN == "" {
		t.Skip("TEST_RESOURCE_ARN not set")
	}

	cfg, err := awsauth.NewAWSConfig(context.Background(), os.Getenv("AWS_REGION"), os.Getenv("AWS_PROFILE"), "")
	require.NoError(t, err)

	client := tagging.New(cfg)
	tags, err := client.ResourceTags(resourceARN)
	require.NoError(t, err)
	t.Logf("found %d tags", len(tags))
}
