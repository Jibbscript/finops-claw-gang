package activities

import (
	"context"
	"fmt"

	"github.com/finops-claw-gang/finops-go/internal/analysis"
	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/executor"
	"github.com/finops-claw-gang/finops-go/internal/ratelimit"
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

// TenantDeps provides per-tenant Cost and Infra clients.
// Implemented by connectors.TenantClientFactory; defined here to avoid import cycles.
type TenantDeps interface {
	CostClient(ctx context.Context, tenant domain.TenantContext) (CostDeps, error)
	InfraClient(ctx context.Context, tenant domain.TenantContext) (InfraDeps, error)
}

// Activities holds the dependencies for all Temporal activities.
// Each method is registered as a Temporal activity.
// When Tenants is non-nil, per-tenant clients are created dynamically;
// otherwise the static Cost/Infra deps are used (stub mode backward compat).
type Activities struct {
	Cost     CostDeps
	Infra    InfraDeps
	KubeCost triage.KubeCostQuerier
	AWSDoc   AWSDocDeps
	Executor *executor.Executor
	Tenants  TenantDeps              // nil in stub mode
	Budget   *ratelimit.ActivityBudget // nil = no budget enforcement
}

// checkBudget enforces per-tenant activity budgets when configured.
func (a *Activities) checkBudget(tenantID, activityName string) error {
	if a.Budget == nil {
		return nil
	}
	if err := a.Budget.Check(tenantID, activityName); err != nil {
		return err
	}
	a.Budget.Record(tenantID, activityName)
	return nil
}

// resolveCost returns per-tenant cost client if available, otherwise the static one.
func (a *Activities) resolveCost(ctx context.Context, tenant domain.TenantContext) (CostDeps, error) {
	if a.Tenants != nil && tenant.IAMRoleARN != "" {
		return a.Tenants.CostClient(ctx, tenant)
	}
	return a.Cost, nil
}

// resolveInfra returns per-tenant infra client if available, otherwise the static one.
func (a *Activities) resolveInfra(ctx context.Context, tenant domain.TenantContext) (InfraDeps, error) {
	if a.Tenants != nil && tenant.IAMRoleARN != "" {
		return a.Tenants.InfraClient(ctx, tenant)
	}
	return a.Infra, nil
}

// TriageAnomaly classifies a cost anomaly using deterministic evidence checks.
func (a *Activities) TriageAnomaly(ctx context.Context, in TriageInput) (TriageOutput, error) {
	if err := a.checkBudget(in.Tenant.TenantID, "TriageAnomaly"); err != nil {
		return TriageOutput{}, err
	}
	cost, err := a.resolveCost(ctx, in.Tenant)
	if err != nil {
		return TriageOutput{}, fmt.Errorf("triage activity: resolve cost: %w", err)
	}
	infra, err := a.resolveInfra(ctx, in.Tenant)
	if err != nil {
		return TriageOutput{}, fmt.Errorf("triage activity: resolve infra: %w", err)
	}
	result, err := triage.Triage(ctx, in.Anomaly, cost, infra, a.KubeCost, a.AWSDoc, in.WindowStart, in.WindowEnd)
	if err != nil {
		return TriageOutput{}, fmt.Errorf("triage activity: %w", err)
	}
	return TriageOutput{Result: result}, nil
}

// PlanActions runs the deterministic analysis planner and returns recommended actions.
func (a *Activities) PlanActions(ctx context.Context, in PlanActionsInput) (PlanActionsOutput, error) {
	if err := a.checkBudget(in.Tenant.TenantID, "PlanActions"); err != nil {
		return PlanActionsOutput{}, err
	}
	cost, err := a.resolveCost(ctx, in.Tenant)
	if err != nil {
		return PlanActionsOutput{}, fmt.Errorf("plan actions activity: resolve cost: %w", err)
	}
	result, err := analysis.AnalyzeAndRecommend(in.AccountID, in.Service, in.WindowStart, in.WindowEnd, cost)
	if err != nil {
		return PlanActionsOutput{}, fmt.Errorf("plan actions activity: %w", err)
	}
	return PlanActionsOutput{Result: result}, nil
}

// ExecuteActions gathers resource tags and runs the executor.
// Tags are fetched inside the activity boundary (I/O belongs here, not in the workflow).
func (a *Activities) ExecuteActions(ctx context.Context, in ExecuteActionsInput) (ExecuteActionsOutput, error) {
	if err := a.checkBudget(in.Tenant.TenantID, "ExecuteActions"); err != nil {
		return ExecuteActionsOutput{}, err
	}
	infra, err := a.resolveInfra(ctx, in.Tenant)
	if err != nil {
		return ExecuteActionsOutput{}, fmt.Errorf("execute activity: resolve infra: %w", err)
	}

	tagsByARN := make(map[string]map[string]string)
	for _, action := range in.Actions {
		if action.TargetResource == "" {
			continue
		}
		tags, err := infra.ResourceTags(action.TargetResource)
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
func (a *Activities) VerifyOutcome(ctx context.Context, in VerifyOutcomeInput) (VerifyOutcomeOutput, error) {
	if err := a.checkBudget(in.Tenant.TenantID, "VerifyOutcome"); err != nil {
		return VerifyOutcomeOutput{}, err
	}
	cost, err := a.resolveCost(ctx, in.Tenant)
	if err != nil {
		return VerifyOutcomeOutput{}, fmt.Errorf("verify activity: resolve cost: %w", err)
	}
	result, err := verifier.Verify(in.Service, in.AccountID, cost, in.WindowStart, in.WindowEnd)
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
