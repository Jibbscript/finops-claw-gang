package costexplorer

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ce "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCEAPI struct {
	costAndUsageOut *ce.GetCostAndUsageOutput
	riCoverageOut   *ce.GetReservationCoverageOutput
	spCoverageOut   *ce.GetSavingsPlansCoverageOutput
	costAndUsageErr error
	riCoverageErr   error
	spCoverageErr   error
}

func (m *mockCEAPI) GetCostAndUsage(_ context.Context, _ *ce.GetCostAndUsageInput, _ ...func(*ce.Options)) (*ce.GetCostAndUsageOutput, error) {
	return m.costAndUsageOut, m.costAndUsageErr
}

func (m *mockCEAPI) GetReservationCoverage(_ context.Context, _ *ce.GetReservationCoverageInput, _ ...func(*ce.Options)) (*ce.GetReservationCoverageOutput, error) {
	return m.riCoverageOut, m.riCoverageErr
}

func (m *mockCEAPI) GetSavingsPlansCoverage(_ context.Context, _ *ce.GetSavingsPlansCoverageInput, _ ...func(*ce.Options)) (*ce.GetSavingsPlansCoverageOutput, error) {
	return m.spCoverageOut, m.spCoverageErr
}

func TestGetCostTimeseries(t *testing.T) {
	mock := &mockCEAPI{
		costAndUsageOut: &ce.GetCostAndUsageOutput{
			ResultsByTime: []cetypes.ResultByTime{
				{
					TimePeriod: &cetypes.DateInterval{
						Start: aws.String("2024-01-01"),
						End:   aws.String("2024-01-02"),
					},
					Total: map[string]cetypes.MetricValue{
						"UnblendedCost": {Amount: aws.String("100.50")},
					},
				},
				{
					TimePeriod: &cetypes.DateInterval{
						Start: aws.String("2024-01-02"),
						End:   aws.String("2024-01-03"),
					},
					Total: map[string]cetypes.MetricValue{
						"UnblendedCost": {Amount: aws.String("80.25")},
					},
				},
			},
		},
	}

	client := NewFromAPI(mock)
	result, err := client.GetCostTimeseries("EC2", "123456789012", "2024-01-01", "2024-01-03")
	require.NoError(t, err)

	savings, ok := result["observed_savings_daily"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 20.25, savings, 0.01) // 100.50 - 80.25

	points, ok := result["points"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, points, 2)
	assert.Equal(t, "2024-01-01", points[0]["start"])
	assert.InDelta(t, 100.50, points[0]["amount"].(float64), 0.01)
}

func TestGetCostTimeseries_SinglePoint(t *testing.T) {
	mock := &mockCEAPI{
		costAndUsageOut: &ce.GetCostAndUsageOutput{
			ResultsByTime: []cetypes.ResultByTime{
				{
					TimePeriod: &cetypes.DateInterval{
						Start: aws.String("2024-01-01"),
						End:   aws.String("2024-01-02"),
					},
					Total: map[string]cetypes.MetricValue{
						"UnblendedCost": {Amount: aws.String("100.00")},
					},
				},
			},
		},
	}

	client := NewFromAPI(mock)
	result, err := client.GetCostTimeseries("EC2", "123456789012", "2024-01-01", "2024-01-02")
	require.NoError(t, err)

	savings := result["observed_savings_daily"].(float64)
	assert.Equal(t, 0.0, savings)
}

func TestGetRICoverage(t *testing.T) {
	mock := &mockCEAPI{
		riCoverageOut: &ce.GetReservationCoverageOutput{
			CoveragesByTime: []cetypes.CoverageByTime{
				{Total: &cetypes.Coverage{
					CoverageHours: &cetypes.CoverageHours{
						CoverageHoursPercentage: aws.String("80.0"),
					},
				}},
				{Total: &cetypes.Coverage{
					CoverageHours: &cetypes.CoverageHours{
						CoverageHoursPercentage: aws.String("75.0"),
					},
				}},
			},
		},
	}

	client := NewFromAPI(mock)
	result, err := client.GetRICoverage("123456789012", "2024-01-01", "2024-01-08")
	require.NoError(t, err)

	delta := result["coverage_delta"].(float64)
	assert.InDelta(t, -5.0, delta, 0.01) // 75.0 - 80.0
}

func TestGetRICoverage_SinglePeriod(t *testing.T) {
	mock := &mockCEAPI{
		riCoverageOut: &ce.GetReservationCoverageOutput{
			CoveragesByTime: []cetypes.CoverageByTime{
				{Total: &cetypes.Coverage{
					CoverageHours: &cetypes.CoverageHours{
						CoverageHoursPercentage: aws.String("80.0"),
					},
				}},
			},
		},
	}

	client := NewFromAPI(mock)
	result, err := client.GetRICoverage("123456789012", "2024-01-01", "2024-01-02")
	require.NoError(t, err)
	assert.Equal(t, 0.0, result["coverage_delta"].(float64))
}

func TestGetSPCoverage(t *testing.T) {
	mock := &mockCEAPI{
		spCoverageOut: &ce.GetSavingsPlansCoverageOutput{
			SavingsPlansCoverages: []cetypes.SavingsPlansCoverage{
				{Coverage: &cetypes.SavingsPlansCoverageData{
					CoveragePercentage: aws.String("90.0"),
				}},
				{Coverage: &cetypes.SavingsPlansCoverageData{
					CoveragePercentage: aws.String("85.0"),
				}},
			},
		},
	}

	client := NewFromAPI(mock)
	result, err := client.GetSPCoverage("123456789012", "2024-01-01", "2024-01-08")
	require.NoError(t, err)

	delta := result["coverage_delta"].(float64)
	assert.InDelta(t, -5.0, delta, 0.01) // 85.0 - 90.0
}
