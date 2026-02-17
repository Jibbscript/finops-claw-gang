// Package queues defines per-queue worker configuration for task-queue partitioning.
package queues

import (
	"fmt"
	"strings"

	"go.temporal.io/sdk/worker"

	"github.com/finops-claw-gang/finops-go/internal/temporal/versioning"
)

// QueueConfig holds worker options for a single task queue.
type QueueConfig struct {
	Name    string
	Options worker.Options
}

// DefaultConfigs returns the standard per-queue worker options.
//
//   - QueueAnomaly: stateful lifecycle workflows, generous concurrency
//   - QueueDetect: read-heavy detection, higher concurrency
//   - QueueExec: restricted writes, tight concurrency
func DefaultConfigs() map[string]QueueConfig {
	return map[string]QueueConfig{
		versioning.QueueAnomaly: {
			Name: versioning.QueueAnomaly,
			Options: worker.Options{
				MaxConcurrentActivityExecutionSize:     10,
				MaxConcurrentWorkflowTaskExecutionSize: 10,
			},
		},
		versioning.QueueDetect: {
			Name: versioning.QueueDetect,
			Options: worker.Options{
				MaxConcurrentActivityExecutionSize:     20,
				MaxConcurrentWorkflowTaskExecutionSize: 5,
			},
		},
		versioning.QueueExec: {
			Name: versioning.QueueExec,
			Options: worker.Options{
				MaxConcurrentActivityExecutionSize:     3,
				MaxConcurrentWorkflowTaskExecutionSize: 1,
			},
		},
	}
}

// ParseQueues parses a comma-separated queue list (e.g. "anomaly,exec")
// into a set of queue names. Accepts both short names ("anomaly") and
// full names ("finops-anomaly"). Returns an error for unknown queues.
func ParseQueues(raw string) ([]string, error) {
	if raw == "" {
		return []string{versioning.QueueAnomaly}, nil
	}

	shortNames := map[string]string{
		"anomaly": versioning.QueueAnomaly,
		"detect":  versioning.QueueDetect,
		"exec":    versioning.QueueExec,
	}
	fullNames := map[string]bool{
		versioning.QueueAnomaly: true,
		versioning.QueueDetect:  true,
		versioning.QueueExec:    true,
	}

	seen := make(map[string]bool)
	var result []string
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		// Resolve short name to full name.
		if full, ok := shortNames[name]; ok {
			name = full
		}
		if !fullNames[name] {
			return nil, fmt.Errorf("unknown queue %q", name)
		}
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	if len(result) == 0 {
		return []string{versioning.QueueAnomaly}, nil
	}
	return result, nil
}
