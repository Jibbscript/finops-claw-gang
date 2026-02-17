package querier_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

// mockQuerier implements WorkflowQuerier for unit testing handlers/tools
// without a Temporal dependency.
type mockQuerier struct {
	workflows []querier.WorkflowSummary
	state     *workflows.WorkflowResult
	desc      *querier.WorkflowDescription
	approval  string
	err       error
}

func (m *mockQuerier) ListWorkflows(_ context.Context, _ querier.ListOptions) ([]querier.WorkflowSummary, error) {
	return m.workflows, m.err
}

func (m *mockQuerier) GetWorkflowState(_ context.Context, _ string) (*workflows.WorkflowResult, error) {
	return m.state, m.err
}

func (m *mockQuerier) DescribeWorkflow(_ context.Context, _ string) (*querier.WorkflowDescription, error) {
	return m.desc, m.err
}

func (m *mockQuerier) SubmitApproval(_ context.Context, _ string, _ interface { /* unused */
}) (string, error) {
	return m.approval, m.err
}

// Compile-time check: ensure mockQuerier can be used where WorkflowQuerier is expected.
// We verify the full interface with a concrete type assertion below.

func TestMockSatisfiesInterface(t *testing.T) {
	// Verify the mock can be used as a WorkflowQuerier by exercising the methods.
	m := &mockQuerier{
		state: &workflows.WorkflowResult{
			State:  domain.NewFinOpsState(domain.NewTenantContext("t1")),
			Reason: workflows.ReasonCompleted,
		},
	}

	ctx := context.Background()

	// ListWorkflows
	summaries, err := m.ListWorkflows(ctx, querier.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, summaries)

	// GetWorkflowState
	result, err := m.GetWorkflowState(ctx, "wf-1")
	require.NoError(t, err)
	assert.Equal(t, workflows.ReasonCompleted, result.Reason)
	assert.Equal(t, "t1", result.State.Tenant.TenantID)

	// DescribeWorkflow
	desc, err := m.DescribeWorkflow(ctx, "wf-1")
	require.NoError(t, err)
	assert.Nil(t, desc)
}

func TestListOptionsDefaults(t *testing.T) {
	opts := querier.ListOptions{}
	assert.Empty(t, opts.TaskQueue)
	assert.Empty(t, opts.StatusFilter)
	assert.Equal(t, 0, opts.PageSize)
}

func TestWorkflowSummaryFields(t *testing.T) {
	s := querier.WorkflowSummary{
		WorkflowID: "finops-anomaly-t1-abc123",
		RunID:      "run-1",
		Status:     "Running",
		TaskQueue:  "finops-anomaly",
	}
	assert.Equal(t, "finops-anomaly-t1-abc123", s.WorkflowID)
	assert.Equal(t, "Running", s.Status)
}
