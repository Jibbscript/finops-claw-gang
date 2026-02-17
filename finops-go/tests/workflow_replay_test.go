package tests

import (
	"os"
	"testing"

	"go.temporal.io/sdk/worker"

	"github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
)

// TestReplayAnomalyLifecycle validates workflow determinism by replaying a recorded history.
//
// To generate the history file:
//  1. Start Temporal + worker in stub mode
//  2. Trigger a workflow: temporal workflow start --task-queue finops-anomaly --type AnomalyLifecycleWorkflow --input '...'
//  3. Export: temporal workflow show --workflow-id WID -o json > tests/testdata/anomaly_lifecycle_history.json
//
// The test skips if the history file doesn't exist (e.g., CI without a Temporal server).
func TestReplayAnomalyLifecycle(t *testing.T) {
	const historyFile = "testdata/anomaly_lifecycle_history.json"

	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		t.Skipf("skipping replay test: %s not found (generate via temporal workflow show)", historyFile)
	}

	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(workflows.AnomalyLifecycleWorkflow)

	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, historyFile)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
}
