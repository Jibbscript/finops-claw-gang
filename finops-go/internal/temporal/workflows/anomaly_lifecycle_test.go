package workflows_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

type AnomalyLifecycleSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *AnomalyLifecycleSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	// Register activity struct so string-based OnActivity mocks work.
	s.env.RegisterActivity(&activities.Activities{})
}

func (s *AnomalyLifecycleSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *AnomalyLifecycleSuite) baseInput() workflows.WorkflowInput {
	return workflows.WorkflowInput{
		Tenant: domain.NewTenantContext("tenant-1"),
		Anomaly: &domain.CostAnomaly{
			AnomalyID:    "anom-1",
			Service:      "EC2",
			AccountID:    "123456789012",
			DeltaDollars: 750,
			DeltaPercent: 25,
		},
		WindowStart: "2026-02-01",
		WindowEnd:   "2026-02-16",
	}
}

// 1. HappyPath_AutoApproved: low risk auto-approved, all 4 activities called
func (s *AnomalyLifecycleSuite) TestHappyPath_AutoApproved() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(activities.TriageOutput{
		Result: domain.TriageResult{
			Category:   domain.CategoryDeployRelated,
			Severity:   domain.SeverityMedium,
			Confidence: 0.7,
			Summary:    "deploy correlated",
		},
	}, nil)

	action := domain.NewRecommendedAction("create alert", "create_budget_alert", domain.RiskLow, "disable alert")
	s.env.OnActivity("PlanActions", testAnyCtx, testAnyInput).Return(activities.PlanActionsOutput{
		Result: domain.AnalysisResult{
			RootCauseNarrative: "test narrative",
			RecommendedActions: []domain.RecommendedAction{action},
		},
	}, nil)

	s.env.OnActivity("ExecuteActions", testAnyCtx, testAnyInput).Return(activities.ExecuteActionsOutput{
		Results: []domain.ExecutionResult{{
			ActionID: action.ActionID,
			Success:  true,
		}},
	}, nil)

	s.env.OnActivity("VerifyOutcome", testAnyCtx, testAnyInput).Return(activities.VerifyOutcomeOutput{
		Result: domain.VerificationResult{
			CostReductionObserved: true,
			ServiceHealthOK:       true,
			Recommendation:        domain.RecommendClose,
		},
	}, nil)

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonCompleted, result.Reason)
	s.Equal(domain.ApprovalAutoApproved, result.State.Approval)
	s.NotNil(result.State.Verification)
}

// 2. ExpectedGrowthEarlyExit: conf >= 0.85, only triage called
func (s *AnomalyLifecycleSuite) TestExpectedGrowthEarlyExit() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(activities.TriageOutput{
		Result: domain.TriageResult{
			Category:   domain.CategoryExpectedGrowth,
			Severity:   domain.SeverityLow,
			Confidence: 0.9,
			Summary:    "usage tracks cost",
		},
	}, nil)

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonExpectedGrowthHighConfidence, result.Reason)
	s.NotNil(result.State.Triage)
	s.Nil(result.State.Analysis)
}

// 3. NoActionsEarlyExit: empty actions, triage + plan called
func (s *AnomalyLifecycleSuite) TestNoActionsEarlyExit() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(activities.TriageOutput{
		Result: domain.TriageResult{
			Category:   domain.CategoryUnknown,
			Severity:   domain.SeverityLow,
			Confidence: 0.4,
		},
	}, nil)

	s.env.OnActivity("PlanActions", testAnyCtx, testAnyInput).Return(activities.PlanActionsOutput{
		Result: domain.AnalysisResult{
			RecommendedActions: []domain.RecommendedAction{},
		},
	}, nil)

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonNoActions, result.Reason)
}

// 4. PolicyDenied: critical risk
func (s *AnomalyLifecycleSuite) TestPolicyDenied() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(activities.TriageOutput{
		Result: domain.TriageResult{
			Category:   domain.CategoryConfigDrift,
			Severity:   domain.SeverityCritical,
			Confidence: 0.8,
		},
	}, nil)

	critAction := domain.NewRecommendedAction("destroy infra", "terminate_instances", domain.RiskCritical, "relaunch")
	s.env.OnActivity("PlanActions", testAnyCtx, testAnyInput).Return(activities.PlanActionsOutput{
		Result: domain.AnalysisResult{
			RecommendedActions: []domain.RecommendedAction{critAction},
		},
	}, nil)

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonPolicyDenied, result.Reason)
	s.Equal(domain.ApprovalDenied, result.State.Approval)
}

// 5. HIL_Approved: medium risk, human approves
func (s *AnomalyLifecycleSuite) TestHIL_Approved() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(activities.TriageOutput{
		Result: domain.TriageResult{
			Category:   domain.CategoryConfigDrift,
			Severity:   domain.SeverityMedium,
			Confidence: 0.75,
		},
	}, nil)

	medAction := domain.NewRecommendedAction("resize instance", "modify_instance", domain.RiskMedium, "revert instance type")
	s.env.OnActivity("PlanActions", testAnyCtx, testAnyInput).Return(activities.PlanActionsOutput{
		Result: domain.AnalysisResult{
			RecommendedActions: []domain.RecommendedAction{medAction},
		},
	}, nil)

	s.env.OnActivity("ExecuteActions", testAnyCtx, testAnyInput).Return(activities.ExecuteActionsOutput{
		Results: []domain.ExecutionResult{{ActionID: medAction.ActionID, Success: true}},
	}, nil)

	s.env.OnActivity("VerifyOutcome", testAnyCtx, testAnyInput).Return(activities.VerifyOutcomeOutput{
		Result: domain.VerificationResult{
			CostReductionObserved: true,
			ServiceHealthOK:       true,
			Recommendation:        domain.RecommendClose,
		},
	}, nil)

	// Simulate human approval after a short delay
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflowNoRejection(workflows.UpdateNameApproval, "test-update-id", s.T(),
			activities.ApprovalResponse{
				Approved: true,
				By:       "ops-engineer",
			})
	}, 1*time.Second)

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonCompleted, result.Reason)
	s.Equal(domain.ApprovalApproved, result.State.Approval)
}

// 6. HIL_Denied: medium risk, human denies
func (s *AnomalyLifecycleSuite) TestHIL_Denied() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(activities.TriageOutput{
		Result: domain.TriageResult{
			Category:   domain.CategoryConfigDrift,
			Severity:   domain.SeverityMedium,
			Confidence: 0.75,
		},
	}, nil)

	medAction := domain.NewRecommendedAction("resize instance", "modify_instance", domain.RiskMedium, "revert")
	s.env.OnActivity("PlanActions", testAnyCtx, testAnyInput).Return(activities.PlanActionsOutput{
		Result: domain.AnalysisResult{
			RecommendedActions: []domain.RecommendedAction{medAction},
		},
	}, nil)

	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflowNoRejection(workflows.UpdateNameApproval, "test-deny-id", s.T(),
			activities.ApprovalResponse{
				Approved: false,
				By:       "ops-lead",
				Reason:   "not safe right now",
			})
	}, 1*time.Second)

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonHumanDenied, result.Reason)
	s.Equal(domain.ApprovalDenied, result.State.Approval)
}

// 7. HIL_Timeout: no response in 24h
func (s *AnomalyLifecycleSuite) TestHIL_Timeout() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(activities.TriageOutput{
		Result: domain.TriageResult{
			Category:   domain.CategoryConfigDrift,
			Severity:   domain.SeverityMedium,
			Confidence: 0.75,
		},
	}, nil)

	medAction := domain.NewRecommendedAction("resize", "modify_instance", domain.RiskMedium, "revert")
	s.env.OnActivity("PlanActions", testAnyCtx, testAnyInput).Return(activities.PlanActionsOutput{
		Result: domain.AnalysisResult{
			RecommendedActions: []domain.RecommendedAction{medAction},
		},
	}, nil)

	// No callback registered -- timer fires after 24h of workflow time
	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonApprovalTimedOut, result.Reason)
	s.Equal(domain.ApprovalTimedOut, result.State.Approval)
}

// 8. TriageActivityError: activity fails
func (s *AnomalyLifecycleSuite) TestTriageActivityError() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(
		activities.TriageOutput{}, fmt.Errorf("cost explorer unavailable"))

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonTriageError, result.Reason)
	s.NotNil(result.State.Error)
}

// 9. ExecutionActivityError: activity fails
func (s *AnomalyLifecycleSuite) TestExecutionActivityError() {
	input := s.baseInput()

	s.env.OnActivity("TriageAnomaly", testAnyCtx, testAnyInput).Return(activities.TriageOutput{
		Result: domain.TriageResult{
			Category:   domain.CategoryDeployRelated,
			Severity:   domain.SeverityMedium,
			Confidence: 0.7,
		},
	}, nil)

	action := domain.NewRecommendedAction("create alert", "create_budget_alert", domain.RiskLow, "disable")
	s.env.OnActivity("PlanActions", testAnyCtx, testAnyInput).Return(activities.PlanActionsOutput{
		Result: domain.AnalysisResult{
			RecommendedActions: []domain.RecommendedAction{action},
		},
	}, nil)

	s.env.OnActivity("ExecuteActions", testAnyCtx, testAnyInput).Return(
		activities.ExecuteActionsOutput{}, fmt.Errorf("executor safety gate failure"))

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonExecutionError, result.Reason)
	s.NotNil(result.State.Error)
}

// NoAnomaly: nil anomaly input
func (s *AnomalyLifecycleSuite) TestNoAnomaly() {
	input := workflows.WorkflowInput{
		Tenant:  domain.NewTenantContext("tenant-1"),
		Anomaly: nil,
	}

	s.env.ExecuteWorkflow(workflows.AnomalyLifecycleWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.WorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(workflows.ReasonNoAnomaly, result.Reason)
}

func TestAnomalyLifecycleSuite(t *testing.T) {
	suite.Run(t, new(AnomalyLifecycleSuite))
}
