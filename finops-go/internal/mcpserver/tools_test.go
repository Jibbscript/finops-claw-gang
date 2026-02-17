package mcpserver_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/mcpserver"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

type stubQuerier struct {
	workflows []querier.WorkflowSummary
	state     *workflows.WorkflowResult
	desc      *querier.WorkflowDescription
	approval  string
	err       error
}

func (s *stubQuerier) ListWorkflows(_ context.Context, _ querier.ListOptions) ([]querier.WorkflowSummary, error) {
	return s.workflows, s.err
}

func (s *stubQuerier) GetWorkflowState(_ context.Context, _ string) (*workflows.WorkflowResult, error) {
	return s.state, s.err
}

func (s *stubQuerier) DescribeWorkflow(_ context.Context, _ string) (*querier.WorkflowDescription, error) {
	return s.desc, s.err
}

func (s *stubQuerier) SubmitApproval(_ context.Context, _ string, _ activities.ApprovalResponse) (string, error) {
	return s.approval, s.err
}

func TestRegisterTools(t *testing.T) {
	q := &stubQuerier{
		state: &workflows.WorkflowResult{
			State: domain.NewFinOpsState(domain.NewTenantContext("t1")),
		},
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	mcpserver.RegisterTools(server, q)

	// Verify it compiles and registers without panic.
	assert.NotNil(t, server)
}
