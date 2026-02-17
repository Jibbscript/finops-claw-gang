package verifier

import (
	"fmt"
	"time"

	"github.com/finops-claw-gang/finops-go/internal/domain"
)

// CostChecker provides cost timeseries data for post-execution verification.
type CostChecker interface {
	GetCostTimeseries(service, accountID, startDate, endDate string) (map[string]any, error)
}

// Verify performs post-execution verification by checking service health and
// observed cost reduction, then recommends close, rollback, or monitor.
func Verify(
	service, accountID string,
	cost CostChecker,
	windowStart, windowEnd string,
) (domain.VerificationResult, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// TODO: production impl should perform real health checks (CloudWatch alarms, error rates, etc.)
	healthDetails := "stub: ok"

	ts, err := cost.GetCostTimeseries(service, accountID, windowStart, windowEnd)
	if err != nil {
		return domain.VerificationResult{}, fmt.Errorf("verifier: get cost timeseries: %w", err)
	}

	observed := extractFloat(ts, "observed_savings_daily")

	if observed > 0 {
		return domain.VerificationResult{
			VerifiedAt:            now,
			CostReductionObserved: true,
			ObservedSavingsDaily:  observed,
			ServiceHealthOK:       true,
			HealthCheckDetails:    healthDetails,
			Recommendation:        domain.RecommendClose,
		}, nil
	}

	return domain.VerificationResult{
		VerifiedAt:            now,
		CostReductionObserved: false,
		ObservedSavingsDaily:  0.0,
		ServiceHealthOK:       true,
		HealthCheckDetails:    healthDetails,
		Recommendation:        domain.RecommendMonitor,
	}, nil
}

// extractFloat safely extracts a float64 from a map[string]any.
// Returns 0.0 if the key is missing or the value is not a numeric type.
func extractFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0.0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0.0
	}
}
