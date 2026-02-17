// Package querier provides read access to Temporal workflow state.
package querier

import "time"

// ListOptions controls filtering for ListWorkflows.
type ListOptions struct {
	// TaskQueue filters by task queue name. Empty means no filter.
	TaskQueue string
	// StatusFilter filters by workflow status (e.g. "Running", "Completed").
	StatusFilter string
	// PageSize limits the number of results.
	PageSize int
}

// WorkflowSummary is a lightweight overview of a workflow execution.
type WorkflowSummary struct {
	WorkflowID string    `json:"workflow_id"`
	RunID      string    `json:"run_id"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	CloseTime  time.Time `json:"close_time,omitempty"`
	TaskQueue  string    `json:"task_queue"`
}

// WorkflowDescription provides detailed info about a workflow execution.
type WorkflowDescription struct {
	WorkflowSummary
	SearchAttributes map[string]any `json:"search_attributes,omitempty"`
}
