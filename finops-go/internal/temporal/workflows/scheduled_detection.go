package workflows

import "go.temporal.io/sdk/workflow"

// DetectionResult is the output of the scheduled detection workflow.
type DetectionResult struct {
	AnomaliesFound int `json:"anomalies_found"`
}

// ScheduledDetectionWorkflow is a stub that will be fleshed out in Phase 3.
// In production it calls DetectAnomalies activity and spawns child
// AnomalyLifecycleWorkflow for each anomaly found.
func ScheduledDetectionWorkflow(ctx workflow.Context) (DetectionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("scheduled detection run (stub)")
	return DetectionResult{AnomaliesFound: 0}, nil
}
