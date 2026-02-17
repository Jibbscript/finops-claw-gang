// Package versioning defines workflow versions and task queue names.
package versioning

const (
	// Workflow versions for determinism tracking.
	AnomalyLifecycleV1 = "anomaly-lifecycle-v1"
	DetectionV1        = "detection-v1"
	AWSDocSweepV1      = "awsdoc-sweep-v1"

	// Task queues. Phase 2 uses QueueAnomaly only; Phase 3 adds
	// permission-isolated queues for detection and execution.
	// Phase 4 adds QueueSweep for aws-doctor sweep workflows.
	QueueAnomaly = "finops-anomaly"
	QueueDetect  = "finops-detect"
	QueueExec    = "finops-exec"
	QueueSweep   = "finops-sweep"
)
