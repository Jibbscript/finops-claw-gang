// Command finops is a CLI tool for triggering and managing FinOps workflows.
//
// Usage:
//
//	finops trigger --tenant T --service S --delta D
//	finops status  --workflow-id WID
//	finops approve --workflow-id WID --by USER
//	finops deny    --workflow-id WID --by USER --reason R
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
	"github.com/finops-claw-gang/finops-go/internal/temporal/versioning"
	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "trigger":
		cmdTrigger(os.Args[2:])
	case "status":
		cmdStatus(os.Args[2:])
	case "approve":
		cmdApprove(os.Args[2:])
	case "deny":
		cmdDeny(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: finops <trigger|status|approve|deny> [flags]")
	os.Exit(1)
}

func dial() client.Client {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("unable to create Temporal client: %v", err)
	}
	return c
}

func cmdTrigger(args []string) {
	fs := flag.NewFlagSet("trigger", flag.ExitOnError)
	tenant := fs.String("tenant", "", "tenant ID (required)")
	service := fs.String("service", "", "AWS service name (required)")
	delta := fs.Float64("delta", 0, "daily dollar delta (required)")
	account := fs.String("account", "123456789012", "AWS account ID")
	windowStart := fs.String("window-start", "2026-02-01", "analysis window start")
	windowEnd := fs.String("window-end", "2026-02-16", "analysis window end")
	_ = fs.Parse(args)

	if *tenant == "" || *service == "" || *delta == 0 {
		fs.Usage()
		os.Exit(1)
	}

	anomaly := domain.NewCostAnomaly()
	anomaly.Service = *service
	anomaly.AccountID = *account
	anomaly.DeltaDollars = *delta

	input := workflows.WorkflowInput{
		Tenant:      domain.NewTenantContext(*tenant),
		Anomaly:     &anomaly,
		WindowStart: *windowStart,
		WindowEnd:   *windowEnd,
	}

	wfID := fmt.Sprintf("finops-anomaly-%s-%s", *tenant, anomaly.AnomalyID)
	c := dial()
	defer c.Close()

	run, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        wfID,
		TaskQueue: versioning.QueueAnomaly,
	}, workflows.AnomalyLifecycleWorkflow, input)
	if err != nil {
		log.Fatalf("failed to start workflow: %v", err)
	}
	fmt.Printf("started workflow %s (run=%s)\n", run.GetID(), run.GetRunID())
}

func cmdStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	wfID := fs.String("workflow-id", "", "workflow ID (required)")
	_ = fs.Parse(args)

	if *wfID == "" {
		fs.Usage()
		os.Exit(1)
	}

	c := dial()
	defer c.Close()

	desc, err := c.DescribeWorkflowExecution(context.Background(), *wfID, "")
	if err != nil {
		log.Fatalf("failed to describe workflow: %v", err)
	}

	data, err := json.MarshalIndent(map[string]any{
		"workflow_id": *wfID,
		"status":      desc.WorkflowExecutionInfo.Status.String(),
		"start_time":  desc.WorkflowExecutionInfo.StartTime,
		"close_time":  desc.WorkflowExecutionInfo.CloseTime,
	}, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal status: %v", err)
	}
	fmt.Println(string(data))
}

func cmdApprove(args []string) {
	fs := flag.NewFlagSet("approve", flag.ExitOnError)
	wfID := fs.String("workflow-id", "", "workflow ID (required)")
	by := fs.String("by", "", "approver identity (required)")
	_ = fs.Parse(args)

	if *wfID == "" || *by == "" {
		fs.Usage()
		os.Exit(1)
	}

	sendUpdate(*wfID, activities.ApprovalResponse{Approved: true, By: *by})
}

func cmdDeny(args []string) {
	fs := flag.NewFlagSet("deny", flag.ExitOnError)
	wfID := fs.String("workflow-id", "", "workflow ID (required)")
	by := fs.String("by", "", "denier identity (required)")
	reason := fs.String("reason", "", "denial reason")
	_ = fs.Parse(args)

	if *wfID == "" || *by == "" {
		fs.Usage()
		os.Exit(1)
	}

	sendUpdate(*wfID, activities.ApprovalResponse{Approved: false, By: *by, Reason: *reason})
}

func sendUpdate(wfID string, resp activities.ApprovalResponse) {
	c := dial()
	defer c.Close()

	handle, err := c.UpdateWorkflow(context.Background(), client.UpdateWorkflowOptions{
		WorkflowID:   wfID,
		UpdateName:   workflows.UpdateNameApproval,
		Args:         []any{resp},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		log.Fatalf("failed to send update: %v", err)
	}

	var result string
	if err := handle.Get(context.Background(), &result); err != nil {
		log.Fatalf("update failed: %v", err)
	}
	fmt.Printf("update result: %s\n", result)
}
