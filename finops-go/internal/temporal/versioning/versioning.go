// Package versioning defines workflow versions and task queue names.
package versioning

const (
	// Workflow versions for determinism tracking.
	AnomalyLifecycleV1 = "anomaly-lifecycle-v1"
	AnomalyLifecycleV2 = "anomaly-lifecycle-v2" // Phase 6: exec-queue-routing + tenant-context
	DetectionV1        = "detection-v1"
	AWSDocSweepV1      = "awsdoc-sweep-v1"

	// Task queues. Phase 2 uses QueueAnomaly only; Phase 3 adds
	// permission-isolated queues for detection and execution.
	// AWSDocSweepWorkflow runs on QueueAnomaly alongside the lifecycle workflow.
	QueueAnomaly = "finops-anomaly"
	QueueDetect  = "finops-detect"
	QueueExec    = "finops-exec"
)
