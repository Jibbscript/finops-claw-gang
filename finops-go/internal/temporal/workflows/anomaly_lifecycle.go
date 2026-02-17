// Package workflows defines the Temporal workflow functions.
package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/policy"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
)

// UpdateNameApproval is the Temporal Update handler name for HIL.
const UpdateNameApproval = "approval"

// HILTimeout is how long the workflow waits for human approval.
const HILTimeout = 24 * time.Hour

// TerminationReason describes why the workflow ended.
type TerminationReason string

const (
	ReasonCompleted                    TerminationReason = "completed"
	ReasonNoAnomaly                    TerminationReason = "no_anomaly"
	ReasonExpectedGrowthHighConfidence TerminationReason = "expected_growth_high_confidence"
	ReasonNoActions                    TerminationReason = "no_actions"
	ReasonPolicyDenied                 TerminationReason = "policy_denied"
	ReasonHumanDenied                  TerminationReason = "human_denied"
	ReasonApprovalTimedOut             TerminationReason = "approval_timed_out"
	ReasonTriageError                  TerminationReason = "triage_error"
	ReasonPlanError                    TerminationReason = "plan_error"
	ReasonExecutionError               TerminationReason = "execution_error"
	ReasonVerifyError                  TerminationReason = "verify_error"
)

// WorkflowInput is the input to the anomaly lifecycle workflow.
type WorkflowInput struct {
	Tenant      domain.TenantContext `json:"tenant"`
	Anomaly     *domain.CostAnomaly  `json:"anomaly"`
	WindowStart string               `json:"window_start"`
	WindowEnd   string               `json:"window_end"`
}

// WorkflowResult is the output of the anomaly lifecycle workflow.
// The workflow returns this on all paths; only infra failures produce
// workflow-level errors.
type WorkflowResult struct {
	State  domain.FinOpsState `json:"state"`
	Reason TerminationReason  `json:"reason"`
}

// AnomalyLifecycleWorkflow is the main Temporal workflow that replaces
// Python's LangGraph StateGraph. The flow is:
//
//	watcher -> triage -> analyst -> hil_gate -> executor -> verifier -> END
//
// Each step may short-circuit to END via early returns.
// Policy runs in-workflow (pure function, no I/O, determinism-safe).
func AnomalyLifecycleWorkflow(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	state := domain.NewFinOpsState(input.Tenant)

	// Activity options: generous timeout, no retry by default (safety first).
	actOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	actCtx := workflow.WithActivityOptions(ctx, actOpts)

	// ------------------------------------------------------------------
	// Watcher: validate anomaly input
	// ------------------------------------------------------------------
	state.CurrentPhase = "watcher"
	if input.Anomaly == nil {
		logger.Info("no anomaly provided, exiting")
		state.ShouldTerminate = true
		return WorkflowResult{State: state, Reason: ReasonNoAnomaly}, nil
	}
	state.Anomaly = input.Anomaly

	// ------------------------------------------------------------------
	// Triage: classify the anomaly
	// ------------------------------------------------------------------
	state.CurrentPhase = "triage"
	var triageOut activities.TriageOutput
	err := workflow.ExecuteActivity(actCtx, "TriageAnomaly", activities.TriageInput{
		Anomaly:     *input.Anomaly,
		WindowStart: input.WindowStart,
		WindowEnd:   input.WindowEnd,
	}).Get(ctx, &triageOut)
	if err != nil {
		errMsg := fmt.Sprintf("triage failed: %v", err)
		state.Error = &errMsg
		state.ShouldTerminate = true
		return WorkflowResult{State: state, Reason: ReasonTriageError}, nil
	}
	state.Triage = &triageOut.Result
	logger.Info("triage complete", "category", triageOut.Result.Category, "confidence", triageOut.Result.Confidence)

	// Early exit: expected growth with high confidence
	if triageOut.Result.Category == domain.CategoryExpectedGrowth && triageOut.Result.Confidence >= 0.85 {
		logger.Info("expected growth with high confidence, exiting early")
		state.ShouldTerminate = true
		return WorkflowResult{State: state, Reason: ReasonExpectedGrowthHighConfidence}, nil
	}

	// ------------------------------------------------------------------
	// Analyst: plan actions
	// ------------------------------------------------------------------
	state.CurrentPhase = "analyst"
	var planOut activities.PlanActionsOutput
	err = workflow.ExecuteActivity(actCtx, "PlanActions", activities.PlanActionsInput{
		AccountID:   input.Anomaly.AccountID,
		Service:     input.Anomaly.Service,
		WindowStart: input.WindowStart,
		WindowEnd:   input.WindowEnd,
	}).Get(ctx, &planOut)
	if err != nil {
		errMsg := fmt.Sprintf("plan actions failed: %v", err)
		state.Error = &errMsg
		state.ShouldTerminate = true
		return WorkflowResult{State: state, Reason: ReasonPlanError}, nil
	}
	state.Analysis = &planOut.Result
	logger.Info("analysis complete", "actions", len(planOut.Result.RecommendedActions))

	// Early exit: no recommended actions
	if len(planOut.Result.RecommendedActions) == 0 {
		logger.Info("no actions recommended, exiting")
		state.ShouldTerminate = true
		return WorkflowResult{State: state, Reason: ReasonNoActions}, nil
	}

	// ------------------------------------------------------------------
	// HIL gate: policy decision + optional human approval
	// ------------------------------------------------------------------
	state.CurrentPhase = "hil_gate"
	pe := policy.NewPolicyEngine()
	decision := pe.Decide(planOut.Result.RecommendedActions)
	state.ApprovalDetails = decision.Details

	switch decision.Approval {
	case domain.ApprovalAutoApproved:
		logger.Info("auto-approved by policy")
		state.Approval = domain.ApprovalAutoApproved

	case domain.ApprovalDenied:
		logger.Info("denied by policy", "details", decision.Details)
		state.Approval = domain.ApprovalDenied
		state.ShouldTerminate = true
		return WorkflowResult{State: state, Reason: ReasonPolicyDenied}, nil

	case domain.ApprovalPending:
		logger.Info("pending human approval", "details", decision.Details)
		state.Approval = domain.ApprovalPending
		approval, err := waitForApproval(ctx)
		if err != nil {
			return WorkflowResult{}, fmt.Errorf("hil gate: %w", err)
		}

		switch approval {
		case domain.ApprovalApproved:
			state.Approval = domain.ApprovalApproved
		case domain.ApprovalDenied:
			state.Approval = domain.ApprovalDenied
			state.ShouldTerminate = true
			return WorkflowResult{State: state, Reason: ReasonHumanDenied}, nil
		case domain.ApprovalTimedOut:
			state.Approval = domain.ApprovalTimedOut
			state.ShouldTerminate = true
			return WorkflowResult{State: state, Reason: ReasonApprovalTimedOut}, nil
		}
	}

	// ------------------------------------------------------------------
	// Executor: run approved actions (no retries for safety)
	// ------------------------------------------------------------------
	state.CurrentPhase = "executor"
	var execOut activities.ExecuteActionsOutput
	err = workflow.ExecuteActivity(actCtx, "ExecuteActions", activities.ExecuteActionsInput{
		Approval: state.Approval,
		Actions:  planOut.Result.RecommendedActions,
	}).Get(ctx, &execOut)
	if err != nil {
		errMsg := fmt.Sprintf("execution failed: %v", err)
		state.Error = &errMsg
		state.ShouldTerminate = true
		return WorkflowResult{State: state, Reason: ReasonExecutionError}, nil
	}
	state.Executions = execOut.Results
	logger.Info("execution complete", "results", len(execOut.Results))

	// ------------------------------------------------------------------
	// Verifier: check outcomes
	// ------------------------------------------------------------------
	state.CurrentPhase = "verifier"
	var verifyOut activities.VerifyOutcomeOutput
	err = workflow.ExecuteActivity(actCtx, "VerifyOutcome", activities.VerifyOutcomeInput{
		Service:     input.Anomaly.Service,
		AccountID:   input.Anomaly.AccountID,
		WindowStart: input.WindowStart,
		WindowEnd:   input.WindowEnd,
	}).Get(ctx, &verifyOut)
	if err != nil {
		errMsg := fmt.Sprintf("verification failed: %v", err)
		state.Error = &errMsg
		state.ShouldTerminate = true
		return WorkflowResult{State: state, Reason: ReasonVerifyError}, nil
	}
	state.Verification = &verifyOut.Result
	state.CurrentPhase = "completed"
	state.ShouldTerminate = true
	logger.Info("workflow completed", "recommendation", verifyOut.Result.Recommendation)

	return WorkflowResult{State: state, Reason: ReasonCompleted}, nil
}

// waitForApproval registers a Temporal Update handler and waits for either
// human approval/denial or a 24-hour timeout, whichever comes first.
func waitForApproval(ctx workflow.Context) (domain.ApprovalStatus, error) {
	logger := workflow.GetLogger(ctx)

	var result domain.ApprovalStatus
	responded := false

	err := workflow.SetUpdateHandlerWithOptions(
		ctx,
		UpdateNameApproval,
		func(ctx workflow.Context, resp activities.ApprovalResponse) (string, error) {
			if responded {
				return "", fmt.Errorf("approval already received")
			}
			responded = true
			if resp.Approved {
				result = domain.ApprovalApproved
				logger.Info("human approved", "by", resp.By)
			} else {
				result = domain.ApprovalDenied
				logger.Info("human denied", "by", resp.By, "reason", resp.Reason)
			}
			return string(result), nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(resp activities.ApprovalResponse) error {
				if resp.By == "" {
					return fmt.Errorf("approval 'by' field is required")
				}
				if responded {
					return fmt.Errorf("approval already received")
				}
				return nil
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("register approval handler: %w", err)
	}

	// Race: approval update vs 24h timeout
	selector := workflow.NewSelector(ctx)

	timer := workflow.NewTimer(ctx, HILTimeout)
	selector.AddFuture(timer, func(f workflow.Future) {
		if !responded {
			result = domain.ApprovalTimedOut
			logger.Info("approval timed out after 24h")
		}
	})

	// The Update handler runs in the Temporal deterministic executor between
	// Select calls, setting `responded = true`. The loop exits when either
	// the handler fires (responded) or the timer expires (ApprovalTimedOut).
	for !responded && result != domain.ApprovalTimedOut {
		selector.Select(ctx)
	}

	return result, nil
}
