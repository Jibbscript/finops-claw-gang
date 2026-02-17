// Package mcpserver exposes FinOps workflow data via MCP tools.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
	"github.com/finops-claw-gang/finops-go/internal/uischema"
)

// RegisterTools registers all FinOps MCP tools on the given server.
func RegisterTools(server *mcp.Server, q querier.WorkflowQuerier) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_anomalies",
			Description: "List recent anomaly workflows with status, service, and cost delta",
		},
		listAnomaliesHandler(q),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_anomaly_state",
			Description: "Get full state and evidence for a specific anomaly workflow",
		},
		getAnomalyStateHandler(q),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_anomaly_ui",
			Description: "Get UI schema (components + actions) for rendering an anomaly workflow",
		},
		getAnomalyUIHandler(q),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "approve_actions",
			Description: "Approve pending workflow actions",
		},
		approveActionsHandler(q),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "deny_actions",
			Description: "Deny pending workflow actions",
		},
		denyActionsHandler(q),
	)
}

type listAnomaliesInput struct {
	Status string `json:"status,omitempty"`
}

func listAnomaliesHandler(q querier.WorkflowQuerier) mcp.ToolHandlerFor[listAnomaliesInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input listAnomaliesInput) (*mcp.CallToolResult, any, error) {
		opts := querier.ListOptions{TaskQueue: "finops-anomaly"}
		if input.Status != "" {
			opts.StatusFilter = input.Status
		}

		workflows, err := q.ListWorkflows(ctx, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("list_anomalies: %w", err)
		}

		return textResult(workflows)
	}
}

type workflowIDInput struct {
	WorkflowID string `json:"workflow_id"`
}

func getAnomalyStateHandler(q querier.WorkflowQuerier) mcp.ToolHandlerFor[workflowIDInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input workflowIDInput) (*mcp.CallToolResult, any, error) {
		if input.WorkflowID == "" {
			return errorResult("workflow_id is required"), nil, nil
		}

		result, err := q.GetWorkflowState(ctx, input.WorkflowID)
		if err != nil {
			return nil, nil, fmt.Errorf("get_anomaly_state: %w", err)
		}

		return textResult(result)
	}
}

func getAnomalyUIHandler(q querier.WorkflowQuerier) mcp.ToolHandlerFor[workflowIDInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input workflowIDInput) (*mcp.CallToolResult, any, error) {
		if input.WorkflowID == "" {
			return errorResult("workflow_id is required"), nil, nil
		}

		result, err := q.GetWorkflowState(ctx, input.WorkflowID)
		if err != nil {
			return nil, nil, fmt.Errorf("get_anomaly_ui: %w", err)
		}

		schema := uischema.Build(result.State)
		return textResult(schema)
	}
}

type approvalInput struct {
	WorkflowID string `json:"workflow_id"`
	By         string `json:"by"`
	Reason     string `json:"reason,omitempty"`
}

func approveActionsHandler(q querier.WorkflowQuerier) mcp.ToolHandlerFor[approvalInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input approvalInput) (*mcp.CallToolResult, any, error) {
		if input.WorkflowID == "" || input.By == "" {
			return errorResult("workflow_id and by are required"), nil, nil
		}

		resp := activities.ApprovalResponse{Approved: true, By: input.By, Reason: input.Reason}
		result, err := q.SubmitApproval(ctx, input.WorkflowID, resp)
		if err != nil {
			return nil, nil, fmt.Errorf("approve_actions: %w", err)
		}

		return textResult(map[string]string{"result": result})
	}
}

func denyActionsHandler(q querier.WorkflowQuerier) mcp.ToolHandlerFor[approvalInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input approvalInput) (*mcp.CallToolResult, any, error) {
		if input.WorkflowID == "" || input.By == "" {
			return errorResult("workflow_id and by are required"), nil, nil
		}

		resp := activities.ApprovalResponse{Approved: false, By: input.By, Reason: input.Reason}
		result, err := q.SubmitApproval(ctx, input.WorkflowID, resp)
		if err != nil {
			return nil, nil, fmt.Errorf("deny_actions: %w", err)
		}

		return textResult(map[string]string{"result": result})
	}
}

func textResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
