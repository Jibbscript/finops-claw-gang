// Package cloudwatch wraps the AWS CloudWatch API to satisfy
// the CloudWatchMetrics portion of triage.InfraQuerier.
package cloudwatch

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cw "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// API is the subset of the CloudWatch client used by this package.
type API interface {
	GetMetricStatistics(ctx context.Context, params *cw.GetMetricStatisticsInput, optFns ...func(*cw.Options)) (*cw.GetMetricStatisticsOutput, error)
}

// Client wraps the CloudWatch API.
type Client struct {
	api API
}

// New creates a CloudWatch client from an AWS config.
func New(cfg aws.Config) *Client {
	return &Client{api: cw.NewFromConfig(cfg)}
}

// NewFromAPI creates a Client from an explicit API implementation (for testing).
func NewFromAPI(api API) *Client {
	return &Client{api: api}
}

// CloudWatchMetrics returns baseline vs current metric values in the fixture-compatible shape:
// {"baseline": float64, "current": float64}
// Baseline: average over 8d ago to 1d ago (7 full days).
// Current: average over 1d ago to now.
func (c *Client) CloudWatchMetrics(resourceID, metricName, namespace string) (map[string]any, error) {
	now := time.Now().UTC()
	oneDayAgo := now.Add(-24 * time.Hour)
	eightDaysAgo := now.Add(-8 * 24 * time.Hour)

	baseline, err := c.getAverage(resourceID, metricName, namespace, eightDaysAgo, oneDayAgo)
	if err != nil {
		return nil, fmt.Errorf("cloudwatch: baseline: %w", err)
	}

	current, err := c.getAverage(resourceID, metricName, namespace, oneDayAgo, now)
	if err != nil {
		return nil, fmt.Errorf("cloudwatch: current: %w", err)
	}

	return map[string]any{
		"baseline": baseline,
		"current":  current,
	}, nil
}

func (c *Client) getAverage(resourceID, metricName, namespace string, start, end time.Time) (float64, error) {
	out, err := c.api.GetMetricStatistics(context.TODO(), &cw.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int32(86400), // 1-day period
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
		Dimensions: []cwtypes.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(resourceID),
			},
		},
	})
	if err != nil {
		return 0, err
	}

	if len(out.Datapoints) == 0 {
		return 0, nil
	}

	var sum float64
	for _, dp := range out.Datapoints {
		if dp.Average != nil {
			sum += *dp.Average
		}
	}
	return sum / float64(len(out.Datapoints)), nil
}
