package costexplorer

import (
	"strconv"

	ce "github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// transformCostTimeseries converts CE GetCostAndUsage output to the fixture shape:
// {"observed_savings_daily": float64, "points": []map[string]any}
func transformCostTimeseries(out *ce.GetCostAndUsageOutput) map[string]any {
	points := make([]map[string]any, 0, len(out.ResultsByTime))
	var first, last float64

	for i, r := range out.ResultsByTime {
		amount := 0.0
		if len(r.Total) > 0 {
			if m, ok := r.Total["UnblendedCost"]; ok && m.Amount != nil {
				amount, _ = strconv.ParseFloat(*m.Amount, 64)
			}
		}

		point := map[string]any{
			"start":  "",
			"end":    "",
			"amount": amount,
		}
		if r.TimePeriod != nil {
			if r.TimePeriod.Start != nil {
				point["start"] = *r.TimePeriod.Start
			}
			if r.TimePeriod.End != nil {
				point["end"] = *r.TimePeriod.End
			}
		}
		points = append(points, point)

		if i == 0 {
			first = amount
		}
		last = amount
	}

	// observed_savings_daily = first day cost - last day cost (positive means savings)
	savings := first - last
	if len(out.ResultsByTime) < 2 {
		savings = 0.0
	}

	return map[string]any{
		"observed_savings_daily": savings,
		"points":                 points,
	}
}

// transformRICoverage converts CE GetReservationCoverage output to:
// {"coverage_delta": float64}
func transformRICoverage(out *ce.GetReservationCoverageOutput) map[string]any {
	if len(out.CoveragesByTime) < 2 {
		return map[string]any{"coverage_delta": 0.0}
	}

	first := extractRICoveragePercent(out.CoveragesByTime[0].Total)
	last := extractRICoveragePercent(out.CoveragesByTime[len(out.CoveragesByTime)-1].Total)

	return map[string]any{
		"coverage_delta": last - first,
	}
}

// extractRICoveragePercent extracts the coverage hours percentage from a Coverage struct.
func extractRICoveragePercent(total *cetypes.Coverage) float64 {
	if total == nil || total.CoverageHours == nil || total.CoverageHours.CoverageHoursPercentage == nil {
		return 0.0
	}
	v, _ := strconv.ParseFloat(*total.CoverageHours.CoverageHoursPercentage, 64)
	return v
}

// transformSPCoverage converts CE GetSavingsPlansCoverage output to:
// {"coverage_delta": float64}
func transformSPCoverage(out *ce.GetSavingsPlansCoverageOutput) map[string]any {
	if len(out.SavingsPlansCoverages) < 2 {
		return map[string]any{"coverage_delta": 0.0}
	}

	first := extractSPCoveragePercent(out.SavingsPlansCoverages[0].Coverage)
	last := extractSPCoveragePercent(out.SavingsPlansCoverages[len(out.SavingsPlansCoverages)-1].Coverage)

	return map[string]any{
		"coverage_delta": last - first,
	}
}

// extractSPCoveragePercent extracts the coverage percentage from SavingsPlansCoverageData.
func extractSPCoveragePercent(data *cetypes.SavingsPlansCoverageData) float64 {
	if data == nil || data.CoveragePercentage == nil {
		return 0.0
	}
	v, _ := strconv.ParseFloat(*data.CoveragePercentage, 64)
	return v
}
