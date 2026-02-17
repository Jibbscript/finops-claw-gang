// Package costexplorer wraps the AWS Cost Explorer API to satisfy
// triage.CostFetcher and verifier.CostChecker interfaces.
package costexplorer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	ce "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// API is the subset of the Cost Explorer client used by this package.
type API interface {
	GetCostAndUsage(ctx context.Context, params *ce.GetCostAndUsageInput, optFns ...func(*ce.Options)) (*ce.GetCostAndUsageOutput, error)
	GetReservationCoverage(ctx context.Context, params *ce.GetReservationCoverageInput, optFns ...func(*ce.Options)) (*ce.GetReservationCoverageOutput, error)
	GetSavingsPlansCoverage(ctx context.Context, params *ce.GetSavingsPlansCoverageInput, optFns ...func(*ce.Options)) (*ce.GetSavingsPlansCoverageOutput, error)
}

// Client wraps the Cost Explorer API.
type Client struct {
	api API
}

// New creates a Cost Explorer client from an AWS config.
func New(cfg aws.Config) *Client {
	return &Client{api: ce.NewFromConfig(cfg)}
}

// NewFromAPI creates a Client from an explicit API implementation (for testing).
func NewFromAPI(api API) *Client {
	return &Client{api: api}
}

// GetCostTimeseries returns daily cost data for a service/account in the fixture-compatible shape:
// {"observed_savings_daily": float64, "points": []map[string]any}
func (c *Client) GetCostTimeseries(service, accountID, startDate, endDate string) (map[string]any, error) {
	input := &ce.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(startDate),
			End:   aws.String(endDate),
		},
		Granularity: cetypes.GranularityDaily,
		Metrics:     []string{"UnblendedCost"},
		Filter: &cetypes.Expression{
			And: []cetypes.Expression{
				{Dimensions: &cetypes.DimensionValues{
					Key:    cetypes.DimensionLinkedAccount,
					Values: []string{accountID},
				}},
				{Dimensions: &cetypes.DimensionValues{
					Key:    cetypes.DimensionService,
					Values: []string{service},
				}},
			},
		},
	}

	out, err := c.api.GetCostAndUsage(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("costexplorer: get cost timeseries: %w", err)
	}

	return transformCostTimeseries(out), nil
}

// GetRICoverage returns RI coverage delta in the fixture-compatible shape:
// {"coverage_delta": float64}
func (c *Client) GetRICoverage(accountID, startDate, endDate string) (map[string]any, error) {
	input := &ce.GetReservationCoverageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(startDate),
			End:   aws.String(endDate),
		},
		Filter: &cetypes.Expression{
			Dimensions: &cetypes.DimensionValues{
				Key:    cetypes.DimensionLinkedAccount,
				Values: []string{accountID},
			},
		},
		Granularity: cetypes.GranularityDaily,
	}

	out, err := c.api.GetReservationCoverage(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("costexplorer: get ri coverage: %w", err)
	}

	return transformRICoverage(out), nil
}

// GetSPCoverage returns Savings Plans coverage delta in the fixture-compatible shape:
// {"coverage_delta": float64}
func (c *Client) GetSPCoverage(accountID, startDate, endDate string) (map[string]any, error) {
	input := &ce.GetSavingsPlansCoverageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(startDate),
			End:   aws.String(endDate),
		},
		Filter: &cetypes.Expression{
			Dimensions: &cetypes.DimensionValues{
				Key:    cetypes.DimensionLinkedAccount,
				Values: []string{accountID},
			},
		},
		Granularity: cetypes.GranularityDaily,
	}

	out, err := c.api.GetSavingsPlansCoverage(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("costexplorer: get sp coverage: %w", err)
	}

	return transformSPCoverage(out), nil
}
