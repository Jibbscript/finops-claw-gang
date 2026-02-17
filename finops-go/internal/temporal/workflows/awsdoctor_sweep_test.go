package workflows_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

type SweepSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *SweepSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.RegisterActivity(&activities.Activities{})
	s.env.RegisterWorkflow(workflows.AnomalyLifecycleWorkflow)
}

func (s *SweepSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *SweepSuite) TestNoAccounts() {
	input := workflows.SweepInput{Accounts: []workflows.SweepAccount{}}

	s.env.ExecuteWorkflow(workflows.AWSDocSweepWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.SweepResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(0, result.AccountsScanned)
	s.Equal(0, result.WasteAnomalies)
	s.Equal(0, result.ChildWorkflowsRun)
}

func (s *SweepSuite) TestWasteAboveThreshold_SpawnsChild() {
	input := workflows.SweepInput{
		Accounts: []workflows.SweepAccount{
			{AccountID: "123456789012", Region: "us-east-1", Profile: "prod"},
		},
	}

	// Waste scan returns findings above the $100 threshold
	s.env.OnActivity("RunAWSDocWaste", testAnyCtx, testAnyInput).Return(activities.AWSDocWasteOutput{
		Findings: []domain.WasteFinding{
			{
				ResourceType:            "EC2",
				ResourceID:              "i-abc123",
				ResourceARN:             "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
				Reason:                  "stopped 30+ days",
				EstimatedMonthlySavings: 150.0,
				Region:                  "us-east-1",
			},
		},
		TotalSavings: 150.0,
	}, nil)

	// The child AnomalyLifecycleWorkflow mock — ctx + input
	s.env.OnWorkflow(workflows.AnomalyLifecycleWorkflow, testAnyCtx, testAnyInput).Return(workflows.WorkflowResult{
		Reason: workflows.ReasonCompleted,
	}, nil)

	s.env.ExecuteWorkflow(workflows.AWSDocSweepWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.SweepResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(1, result.AccountsScanned)
	s.Equal(1, result.WasteAnomalies)
	s.Equal(1, result.ChildWorkflowsRun)
}

func (s *SweepSuite) TestWasteBelowThreshold_NoChild() {
	input := workflows.SweepInput{
		Accounts: []workflows.SweepAccount{
			{AccountID: "123456789012", Region: "us-east-1"},
		},
	}

	s.env.OnActivity("RunAWSDocWaste", testAnyCtx, testAnyInput).Return(activities.AWSDocWasteOutput{
		Findings: []domain.WasteFinding{
			{
				ResourceType:            "ElasticIP",
				ResourceID:              "eipalloc-abc",
				EstimatedMonthlySavings: 3.60,
			},
		},
		TotalSavings: 3.60,
	}, nil)

	s.env.ExecuteWorkflow(workflows.AWSDocSweepWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.SweepResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(1, result.AccountsScanned)
	s.Equal(0, result.WasteAnomalies)
	s.Equal(0, result.ChildWorkflowsRun)
}

func (s *SweepSuite) TestActivityError_ContinuesWithCount() {
	input := workflows.SweepInput{
		Accounts: []workflows.SweepAccount{
			{AccountID: "123456789012", Region: "us-east-1"},
		},
	}

	s.env.OnActivity("RunAWSDocWaste", testAnyCtx, testAnyInput).Return(
		activities.AWSDocWasteOutput{}, fmt.Errorf("aws-doctor binary not found"))

	s.env.ExecuteWorkflow(workflows.AWSDocSweepWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.SweepResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(1, result.AccountsScanned)
	s.Equal(1, result.ScanErrors)
	s.Equal(0, result.WasteAnomalies)
	s.Equal(0, result.ChildWorkflowsRun)
}

func (s *SweepSuite) TestMultipleAccounts_MixedResults() {
	input := workflows.SweepInput{
		Accounts: []workflows.SweepAccount{
			{AccountID: "111111111111", Region: "us-east-1", Profile: "prod1"},
			{AccountID: "222222222222", Region: "us-west-2", Profile: "prod2"},
			{AccountID: "333333333333", Region: "eu-west-1", Profile: "prod3"},
		},
	}

	// Account 1: waste above threshold -> spawns child
	s.env.OnActivity("RunAWSDocWaste", testAnyCtx, testAnyInput).Return(activities.AWSDocWasteOutput{
		Findings: []domain.WasteFinding{
			{ResourceType: "EC2", ResourceID: "i-111", EstimatedMonthlySavings: 200.0},
		},
		TotalSavings: 200.0,
	}, nil).Once()

	// Account 2: waste below threshold -> no child
	s.env.OnActivity("RunAWSDocWaste", testAnyCtx, testAnyInput).Return(activities.AWSDocWasteOutput{
		Findings:     []domain.WasteFinding{{ResourceType: "ElasticIP", EstimatedMonthlySavings: 3.60}},
		TotalSavings: 3.60,
	}, nil).Once()

	// Account 3: scan error -> logged and skipped
	s.env.OnActivity("RunAWSDocWaste", testAnyCtx, testAnyInput).Return(
		activities.AWSDocWasteOutput{}, fmt.Errorf("timeout")).Once()

	// Child workflow for account 1
	s.env.OnWorkflow(workflows.AnomalyLifecycleWorkflow, testAnyCtx, testAnyInput).Return(workflows.WorkflowResult{
		Reason: workflows.ReasonCompleted,
	}, nil)

	s.env.ExecuteWorkflow(workflows.AWSDocSweepWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.SweepResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(3, result.AccountsScanned)
	s.Equal(1, result.WasteAnomalies)
	s.Equal(1, result.ChildWorkflowsRun)
	s.Equal(1, result.ScanErrors)
}

func (s *SweepSuite) TestChildWorkflowFailure_ContinuesCount() {
	input := workflows.SweepInput{
		Accounts: []workflows.SweepAccount{
			{AccountID: "123456789012", Region: "us-east-1"},
		},
	}

	s.env.OnActivity("RunAWSDocWaste", testAnyCtx, testAnyInput).Return(activities.AWSDocWasteOutput{
		Findings: []domain.WasteFinding{
			{ResourceType: "EC2", ResourceID: "i-abc", EstimatedMonthlySavings: 500.0},
		},
		TotalSavings: 500.0,
	}, nil)

	// Child workflow fails
	s.env.OnWorkflow(workflows.AnomalyLifecycleWorkflow, testAnyCtx, testAnyInput).Return(
		workflows.WorkflowResult{}, fmt.Errorf("child workflow error"))

	s.env.ExecuteWorkflow(workflows.AWSDocSweepWorkflow, input)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.SweepResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(1, result.AccountsScanned)
	s.Equal(1, result.WasteAnomalies)
	// Child failed, so ChildWorkflowsRun should not be incremented
	s.Equal(0, result.ChildWorkflowsRun)
}

func TestSweepSuite(t *testing.T) {
	suite.Run(t, new(SweepSuite))
}
