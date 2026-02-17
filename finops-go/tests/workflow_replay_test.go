package tests

// Replay test validates workflow determinism by replaying a recorded history.
//
// Phase 2 bootstrap: the test is a stub that will be activated once we have
// a recorded history JSON in tests/testdata/. To generate:
//
//  1. Run the worker + trigger a workflow via CLI
//  2. Export history: temporal workflow show --workflow-id WID -o json > tests/testdata/anomaly_lifecycle_history.json
//  3. Uncomment the test below.
//
// import (
//     "testing"
//     "go.temporal.io/sdk/worker"
//     "github.com/finops-claw-gang/finops-go/internal/temporal/workflows"
// )
//
// func TestReplayAnomalyLifecycle(t *testing.T) {
//     replayer := worker.NewWorkflowReplayer()
//     replayer.RegisterWorkflow(workflows.AnomalyLifecycleWorkflow)
//     err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "testdata/anomaly_lifecycle_history.json")
//     if err != nil {
//         t.Fatalf("replay failed: %v", err)
//     }
// }
