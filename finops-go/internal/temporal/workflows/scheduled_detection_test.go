package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

type DetectionSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *DetectionSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *DetectionSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *DetectionSuite) TestStubReturnsZero() {
	s.env.ExecuteWorkflow(workflows.ScheduledDetectionWorkflow)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result workflows.DetectionResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(0, result.AnomaliesFound)
}

func TestDetectionSuite(t *testing.T) {
	suite.Run(t, new(DetectionSuite))
}
