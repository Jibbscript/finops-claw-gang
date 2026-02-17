// Package activities defines the Temporal activity I/O structs and the
// Activities implementation that bridges Temporal's serialization boundary
// to the pure-logic packages in internal/.
package activities

import "github.com/finops-claw-gang/finops-go/internal/domain"

// TriageInput is the activity input for anomaly triage.
type TriageInput struct {
	Tenant      domain.TenantContext `json:"tenant,omitempty"`
	Anomaly     domain.CostAnomaly   `json:"anomaly"`
	WindowStart string               `json:"window_start"`
	WindowEnd   string               `json:"window_end"`
}

// TriageOutput is the activity output from anomaly triage.
type TriageOutput struct {
	Result domain.TriageResult `json:"result"`
}

// PlanActionsInput is the activity input for analysis/planning.
type PlanActionsInput struct {
	Tenant      domain.TenantContext `json:"tenant,omitempty"`
	AccountID   string               `json:"account_id"`
	Service     string               `json:"service"`
	WindowStart string               `json:"window_start"`
	WindowEnd   string               `json:"window_end"`
}

// PlanActionsOutput is the activity output from analysis/planning.
type PlanActionsOutput struct {
	Result domain.AnalysisResult `json:"result"`
}

// ExecuteActionsInput is the activity input for action execution.
// Tags are fetched inside the activity boundary, not passed in.
type ExecuteActionsInput struct {
	Tenant   domain.TenantContext       `json:"tenant,omitempty"`
	Approval domain.ApprovalStatus      `json:"approval"`
	Actions  []domain.RecommendedAction `json:"actions"`
}

// ExecuteActionsOutput is the activity output from action execution.
type ExecuteActionsOutput struct {
	Results []domain.ExecutionResult `json:"results"`
}

// VerifyOutcomeInput is the activity input for post-execution verification.
type VerifyOutcomeInput struct {
	Tenant      domain.TenantContext `json:"tenant,omitempty"`
	Service     string               `json:"service"`
	AccountID   string               `json:"account_id"`
	WindowStart string               `json:"window_start"`
	WindowEnd   string               `json:"window_end"`
}

// VerifyOutcomeOutput is the activity output from verification.
type VerifyOutcomeOutput struct {
	Result domain.VerificationResult `json:"result"`
}

// NotifySlackInput is the activity input for Slack notifications.
type NotifySlackInput struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}

// CreateTicketInput is the activity input for ticket creation.
type CreateTicketInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

// CreateTicketOutput is the activity output from ticket creation.
type CreateTicketOutput struct {
	TicketID  string `json:"ticket_id"`
	TicketURL string `json:"ticket_url"`
}

// ApprovalResponse is sent via the Temporal Update handler for HIL.
type ApprovalResponse struct {
	Approved bool   `json:"approved"`
	By       string `json:"by"`
	Reason   string `json:"reason,omitempty"`
}

// AWSDocWasteInput is the activity input for aws-doctor waste scans.
type AWSDocWasteInput struct {
	AccountID string `json:"account_id"`
	Region    string `json:"region"`
	Profile   string `json:"profile"`
}

// AWSDocWasteOutput is the activity output from aws-doctor waste scans.
type AWSDocWasteOutput struct {
	Findings     []domain.WasteFinding `json:"findings"`
	TotalSavings float64               `json:"total_savings"`
}

// AWSDocTrendInput is the activity input for aws-doctor trend analysis.
type AWSDocTrendInput struct {
	Profile string `json:"profile"`
	Region  string `json:"region"`
}

// AWSDocTrendOutput is the activity output from aws-doctor trend analysis.
type AWSDocTrendOutput struct {
	TrendDirection string  `json:"trend_direction"`
	VelocityPct    float64 `json:"velocity_pct"`
}
