// Package versioning defines workflow versions and task queue names.
package versioning

const (
	// Workflow versions for determinism tracking.
	AnomalyLifecycleV1 = "anomaly-lifecycle-v1"
	DetectionV1        = "detection-v1"

	// Task queues. Phase 2 uses QueueAnomaly only; Phase 3 adds
	// permission-isolated queues for detection and execution.
	QueueAnomaly = "finops-anomaly"
	QueueDetect  = "finops-detect"
	QueueExec    = "finops-exec"
)
