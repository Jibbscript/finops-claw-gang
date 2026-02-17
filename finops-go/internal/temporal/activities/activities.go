package activities

import (
	"context"
	"fmt"

	"github.com/finops-claw-gang/finops-go/internal/analysis"
	"github.com/finops-claw-gang/finops-go/internal/executor"
	"github.com/finops-claw-gang/finops-go/internal/triage"
	"github.com/finops-claw-gang/finops-go/internal/verifier"
)

// CostDeps is the union of all cost-related interfaces consumed by activities.
// testutil.StubCost already satisfies this without changes.
type CostDeps interface {
	triage.CostFetcher
	analysis.CostQuerier
	verifier.CostChecker
}

// InfraDeps is the union of infrastructure interfaces consumed by activities.
// testutil.StubInfra already satisfies this without changes.
type InfraDeps interface {
	triage.InfraQuerier
	executor.TagFetcher
}

// AWSDocDeps provides aws-doctor waste query capability to activities.
type AWSDocDeps interface {
	triage.WasteQuerier
}

// Activities holds the dependencies for all Temporal activities.
// Each method is registered as a Temporal activity.
type Activities struct {
	Cost     CostDeps
	Infra    InfraDeps
	KubeCost triage.KubeCostQuerier
	AWSDoc   AWSDocDeps
	Executor *executor.Executor
}

// TriageAnomaly classifies a cost anomaly using deterministic evidence checks.
func (a *Activities) TriageAnomaly(ctx context.Context, in TriageInput) (TriageOutput, error) {
	result, err := triage.Triage(ctx, in.Anomaly, a.Cost, a.Infra, a.KubeCost, a.AWSDoc, in.WindowStart, in.WindowEnd)
	if err != nil {
		return TriageOutput{}, fmt.Errorf("triage activity: %w", err)
	}
	return TriageOutput{Result: result}, nil
}

// PlanActions runs the deterministic analysis planner and returns recommended actions.
func (a *Activities) PlanActions(_ context.Context, in PlanActionsInput) (PlanActionsOutput, error) {
	result, err := analysis.AnalyzeAndRecommend(in.AccountID, in.Service, in.WindowStart, in.WindowEnd, a.Cost)
	if err != nil {
		return PlanActionsOutput{}, fmt.Errorf("plan actions activity: %w", err)
	}
	return PlanActionsOutput{Result: result}, nil
}

// ExecuteActions gathers resource tags and runs the executor.
// Tags are fetched inside the activity boundary (I/O belongs here, not in the workflow).
func (a *Activities) ExecuteActions(_ context.Context, in ExecuteActionsInput) (ExecuteActionsOutput, error) {
	// Build resource tags map from the actions' target resources.
	tagsByARN := make(map[string]map[string]string)
	for _, action := range in.Actions {
		if action.TargetResource == "" {
			continue
		}
		tags, err := a.Infra.ResourceTags(action.TargetResource)
		if err != nil {
			return ExecuteActionsOutput{}, fmt.Errorf("execute activity: fetch tags for %s: %w", action.TargetResource, err)
		}
		tagsByARN[action.TargetResource] = tags
	}

	results, err := a.Executor.ExecuteActions(in.Approval, in.Actions, tagsByARN)
	if err != nil {
		return ExecuteActionsOutput{}, fmt.Errorf("execute activity: %w", err)
	}
	return ExecuteActionsOutput{Results: results}, nil
}

// VerifyOutcome checks service health and observed cost reduction.
func (a *Activities) VerifyOutcome(_ context.Context, in VerifyOutcomeInput) (VerifyOutcomeOutput, error) {
	result, err := verifier.Verify(in.Service, in.AccountID, a.Cost, in.WindowStart, in.WindowEnd)
	if err != nil {
		return VerifyOutcomeOutput{}, fmt.Errorf("verify activity: %w", err)
	}
	return VerifyOutcomeOutput{Result: result}, nil
}

// RunAWSDocWaste runs an aws-doctor waste scan and returns domain-level findings.
func (a *Activities) RunAWSDocWaste(ctx context.Context, in AWSDocWasteInput) (AWSDocWasteOutput, error) {
	if a.AWSDoc == nil {
		return AWSDocWasteOutput{}, fmt.Errorf("aws-doctor not configured")
	}
	findings, err := a.AWSDoc.Waste(ctx, in.AccountID, in.Region, in.Profile)
	if err != nil {
		return AWSDocWasteOutput{}, fmt.Errorf("aws-doctor waste: %w", err)
	}
	var total float64
	for _, f := range findings {
		total += f.EstimatedMonthlySavings
	}
	return AWSDocWasteOutput{Findings: findings, TotalSavings: total}, nil
}

// RunAWSDocTrend runs an aws-doctor trend analysis. Currently stubbed —
// the sweep workflow calls this to enrich evidence but the triage path
// doesn't require it (trend data is supplementary).
func (a *Activities) RunAWSDocTrend(_ context.Context, in AWSDocTrendInput) (AWSDocTrendOutput, error) {
	// Phase 4 stub: real implementation would call Runner.Trend() + MapTrendMetrics().
	return AWSDocTrendOutput{TrendDirection: "stable", VelocityPct: 0}, nil
}

// NotifySlack sends a notification to Slack. Stubbed for Phase 2.
func (a *Activities) NotifySlack(_ context.Context, in NotifySlackInput) error {
	// Phase 3: real Slack webhook call.
	return nil
}

// CreateTicket creates a ticket in the ticketing system. Stubbed for Phase 2.
func (a *Activities) CreateTicket(_ context.Context, in CreateTicketInput) (CreateTicketOutput, error) {
	// Phase 3: real Jira/Linear API call.
	return CreateTicketOutput{
		TicketID:  "STUB-001",
		TicketURL: "https://example.com/tickets/STUB-001",
	}, nil
}
