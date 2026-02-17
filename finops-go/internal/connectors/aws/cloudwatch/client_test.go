package cloudwatch

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cw "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCWAPI struct {
	calls int
	outs  []*cw.GetMetricStatisticsOutput
	err   error
}

func (m *mockCWAPI) GetMetricStatistics(_ context.Context, _ *cw.GetMetricStatisticsInput, _ ...func(*cw.Options)) (*cw.GetMetricStatisticsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	idx := m.calls
	if idx >= len(m.outs) {
		idx = len(m.outs) - 1
	}
	m.calls++
	return m.outs[idx], nil
}

func TestCloudWatchMetrics(t *testing.T) {
	mock := &mockCWAPI{
		outs: []*cw.GetMetricStatisticsOutput{
			// Baseline call (7 days).
			{Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(1000.0)},
				{Average: aws.Float64(1100.0)},
			}},
			// Current call (1 day).
			{Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(1050.0)},
			}},
		},
	}

	client := NewFromAPI(mock)
	result, err := client.CloudWatchMetrics("i-1234", "CPUUtilization", "AWS/EC2")
	require.NoError(t, err)

	// Baseline: (1000 + 1100) / 2 = 1050
	assert.InDelta(t, 1050.0, result["baseline"].(float64), 0.01)
	assert.InDelta(t, 1050.0, result["current"].(float64), 0.01)
	assert.Equal(t, 2, mock.calls)
}

func TestCloudWatchMetrics_NoDatapoints(t *testing.T) {
	mock := &mockCWAPI{
		outs: []*cw.GetMetricStatisticsOutput{
			{Datapoints: nil},
			{Datapoints: nil},
		},
	}

	client := NewFromAPI(mock)
	result, err := client.CloudWatchMetrics("i-1234", "CPUUtilization", "AWS/EC2")
	require.NoError(t, err)

	assert.Equal(t, 0.0, result["baseline"].(float64))
	assert.Equal(t, 0.0, result["current"].(float64))
}
