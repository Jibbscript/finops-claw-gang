package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
)

// WasteSavingsThreshold is the monthly savings threshold (in dollars)
// above which a waste scan spawns a child anomaly lifecycle workflow.
const WasteSavingsThreshold = 100.0

// SweepInput configures which accounts to scan.
type SweepInput struct {
	Accounts []SweepAccount `json:"accounts"`
}

// SweepAccount identifies one AWS account to scan.
type SweepAccount struct {
	AccountID string `json:"account_id"`
	Region    string `json:"region"`
	Profile   string `json:"profile"`
}

// SweepResult summarizes the sweep outcome.
type SweepResult struct {
	AccountsScanned   int `json:"accounts_scanned"`
	WasteAnomalies    int `json:"waste_anomalies"`
	ChildWorkflowsRun int `json:"child_workflows_run"`
}

// AWSDocSweepWorkflow runs aws-doctor waste scans across configured accounts.
// For each account with waste savings above the threshold, it synthesizes a
// CostAnomaly and starts a child AnomalyLifecycleWorkflow.
func AWSDocSweepWorkflow(ctx workflow.Context, input SweepInput) (SweepResult, error) {
	logger := workflow.GetLogger(ctx)
	result := SweepResult{}

	actOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	actCtx := workflow.WithActivityOptions(ctx, actOpts)

	for _, acct := range input.Accounts {
		result.AccountsScanned++

		// Run waste scan
		var wasteOut activities.AWSDocWasteOutput
		err := workflow.ExecuteActivity(actCtx, "RunAWSDocWaste", activities.AWSDocWasteInput{
			AccountID: acct.AccountID,
			Region:    acct.Region,
			Profile:   acct.Profile,
		}).Get(ctx, &wasteOut)
		if err != nil {
			return result, fmt.Errorf("waste scan for %s: %w", acct.AccountID, err)
		}

		logger.Info("waste scan complete",
			"account", acct.AccountID,
			"findings", len(wasteOut.Findings),
			"total_savings", wasteOut.TotalSavings,
		)

		if wasteOut.TotalSavings < WasteSavingsThreshold {
			continue
		}
		result.WasteAnomalies++

		// Synthesize a CostAnomaly for the lifecycle workflow
		anomaly := domain.NewCostAnomaly()
		anomaly.Service = "MultiService"
		anomaly.AccountID = acct.AccountID
		anomaly.Region = acct.Region
		anomaly.DeltaDollars = wasteOut.TotalSavings
		anomaly.DeltaPercent = 0 // waste is absolute, not relative

		childOpts := workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("waste-%s-%s", acct.AccountID, anomaly.AnomalyID),
		}
		childCtx := workflow.WithChildOptions(ctx, childOpts)

		var childResult WorkflowResult
		err = workflow.ExecuteChildWorkflow(childCtx, AnomalyLifecycleWorkflow, WorkflowInput{
			Tenant:  domain.NewTenantContext(acct.AccountID),
			Anomaly: &anomaly,
		}).Get(ctx, &childResult)
		if err != nil {
			logger.Warn("child workflow failed", "account", acct.AccountID, "error", err)
			continue
		}
		result.ChildWorkflowsRun++
		logger.Info("child workflow completed",
			"account", acct.AccountID,
			"reason", childResult.Reason,
		)
	}

	return result, nil
}
