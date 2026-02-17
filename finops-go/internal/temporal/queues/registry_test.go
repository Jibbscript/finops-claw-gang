package queues

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/finops-claw-gang/finops-go/internal/temporal/versioning"
)

func TestDefaultConfigs(t *testing.T) {
	configs := DefaultConfigs()
	assert.Len(t, configs, 3)
	assert.Contains(t, configs, versioning.QueueAnomaly)
	assert.Contains(t, configs, versioning.QueueDetect)
	assert.Contains(t, configs, versioning.QueueExec)

	// Exec queue should have tightest concurrency.
	execCfg := configs[versioning.QueueExec]
	assert.Equal(t, 3, execCfg.Options.MaxConcurrentActivityExecutionSize)
}

func TestParseQueues(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr string
	}{
		{"empty defaults to anomaly", "", []string{versioning.QueueAnomaly}, ""},
		{"short name anomaly", "anomaly", []string{versioning.QueueAnomaly}, ""},
		{"short name exec", "exec", []string{versioning.QueueExec}, ""},
		{"full name", "finops-anomaly", []string{versioning.QueueAnomaly}, ""},
		{"multiple", "anomaly,exec,detect", []string{versioning.QueueAnomaly, versioning.QueueExec, versioning.QueueDetect}, ""},
		{"deduplicate", "anomaly,anomaly", []string{versioning.QueueAnomaly}, ""},
		{"spaces trimmed", " anomaly , exec ", []string{versioning.QueueAnomaly, versioning.QueueExec}, ""},
		{"unknown queue", "bogus", nil, `unknown queue "bogus"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQueues(tt.raw)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
