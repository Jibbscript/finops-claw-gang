package querier

import (
	"context"

	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

// WorkflowQuerier provides read access to workflow state and the ability
// to submit approvals. Used by the HTTP API, AG-UI streamer, and MCP server.
type WorkflowQuerier interface {
	ListWorkflows(ctx context.Context, opts ListOptions) ([]WorkflowSummary, error)
	GetWorkflowState(ctx context.Context, workflowID string) (*workflows.WorkflowResult, error)
	DescribeWorkflow(ctx context.Context, workflowID string) (*WorkflowDescription, error)
	SubmitApproval(ctx context.Context, workflowID string, resp activities.ApprovalResponse) (string, error)
}
