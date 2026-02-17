package querier

import (
	"context"
	"fmt"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

// TemporalQuerier implements WorkflowQuerier using a Temporal client.
type TemporalQuerier struct {
	client client.Client
}

// New creates a TemporalQuerier.
func New(c client.Client) *TemporalQuerier {
	return &TemporalQuerier{client: c}
}

// ListWorkflows lists workflow executions using Temporal's visibility API.
func (q *TemporalQuerier) ListWorkflows(ctx context.Context, opts ListOptions) ([]WorkflowSummary, error) {
	query := ""
	if opts.TaskQueue != "" {
		query = fmt.Sprintf("TaskQueue = %q", opts.TaskQueue)
	}
	if opts.StatusFilter != "" {
		if query != "" {
			query += " AND "
		}
		query += fmt.Sprintf("ExecutionStatus = %q", opts.StatusFilter)
	}

	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	resp, err := q.client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Query:    query,
		PageSize: int32(pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}

	var summaries []WorkflowSummary
	for _, exec := range resp.Executions {
		s := WorkflowSummary{
			WorkflowID: exec.Execution.WorkflowId,
			RunID:      exec.Execution.RunId,
			Status:     exec.Status.String(),
			StartTime:  exec.StartTime.AsTime(),
			TaskQueue:  exec.TaskQueue,
		}
		if exec.CloseTime != nil {
			s.CloseTime = exec.CloseTime.AsTime()
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// GetWorkflowState returns the current workflow result.
// For completed workflows, extracts the result directly.
// For running workflows, uses the Query handler.
func (q *TemporalQuerier) GetWorkflowState(ctx context.Context, workflowID string) (*workflows.WorkflowResult, error) {
	desc, err := q.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, fmt.Errorf("describe workflow: %w", err)
	}

	status := desc.WorkflowExecutionInfo.Status
	if status == enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
		run := q.client.GetWorkflow(ctx, workflowID, "")
		var result workflows.WorkflowResult
		if err := run.Get(ctx, &result); err != nil {
			return nil, fmt.Errorf("get workflow result: %w", err)
		}
		return &result, nil
	}

	if status == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		resp, err := q.client.QueryWorkflow(ctx, workflowID, "", workflows.QueryNameState)
		if err != nil {
			return nil, fmt.Errorf("query workflow state: %w", err)
		}
		var result workflows.WorkflowResult
		if err := resp.Get(&result); err != nil {
			return nil, fmt.Errorf("decode query result: %w", err)
		}
		return &result, nil
	}

	return nil, fmt.Errorf("workflow %s has status %s, cannot read state", workflowID, status)
}

// DescribeWorkflow returns detailed information about a workflow execution.
func (q *TemporalQuerier) DescribeWorkflow(ctx context.Context, workflowID string) (*WorkflowDescription, error) {
	desc, err := q.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, fmt.Errorf("describe workflow: %w", err)
	}

	info := desc.WorkflowExecutionInfo
	wd := &WorkflowDescription{
		WorkflowSummary: WorkflowSummary{
			WorkflowID: info.Execution.WorkflowId,
			RunID:      info.Execution.RunId,
			Status:     info.Status.String(),
			StartTime:  info.StartTime.AsTime(),
			TaskQueue:  info.TaskQueue,
		},
	}
	if info.CloseTime != nil {
		wd.CloseTime = info.CloseTime.AsTime()
	}
	return wd, nil
}

// SubmitApproval sends an approval/denial Update to a running workflow.
func (q *TemporalQuerier) SubmitApproval(ctx context.Context, workflowID string, resp activities.ApprovalResponse) (string, error) {
	handle, err := q.client.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   workflows.UpdateNameApproval,
		Args:         []any{resp},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return "", fmt.Errorf("submit approval: %w", err)
	}

	var result string
	if err := handle.Get(ctx, &result); err != nil {
		return "", fmt.Errorf("get approval result: %w", err)
	}
	return result, nil
}
