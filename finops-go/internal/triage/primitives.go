package triage

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/finops-claw-gang/finops-go/internal/domain"
)

// SeverityFromDelta maps a daily dollar delta to an anomaly severity level.
func SeverityFromDelta(deltaDollarsDaily float64) domain.AnomalySeverity {
	if deltaDollarsDaily >= 5000 {
		return domain.SeverityCritical
	}
	if deltaDollarsDaily >= 1000 {
		return domain.SeverityHigh
	}
	if deltaDollarsDaily >= 200 {
		return domain.SeverityMedium
	}
	return domain.SeverityLow
}

// PctChange computes the percentage change from oldVal to newVal.
// If oldVal is zero, returns 1.0 when newVal != 0 and 0.0 otherwise.
func PctChange(newVal, oldVal float64) float64 {
	if oldVal == 0 {
		if newVal != 0 {
			return 1.0
		}
		return 0.0
	}
	return (newVal - oldVal) / oldVal
}

// float64Ptr returns a pointer to the given float64 value.
func float64Ptr(v float64) *float64 {
	return &v
}

// floatFromMap extracts a float64 from a map[string]any by key,
// returning fallback if the key is missing or cannot be converted.
func floatFromMap(m map[string]any, key string, fallback float64) float64 {
	v, ok := m[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return fallback
		}
		return f
	}
	return fallback
}

// stringFromMap extracts a string from a map[string]any by key,
// returning fallback if the key is missing or not a string.
func stringFromMap(m map[string]any, key string, fallback string) string {
	v, ok := m[key]
	if !ok {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	return s
}

// Triage classifies a cost anomaly using a priority-ordered sequence of
// deterministic evidence checks. No LLM is involved.
//
// Priority order:
//  1. RI/SP commitment coverage drift
//  2. Credits / refunds / fees
//  3. Marketplace charges
//  4. Data transfer spike
//  5. KubeCost namespace allocation shift (kubecost may be nil)
//  6. Deploy correlation
//  7. Expected growth (usage vs cost pct change)
//  8. Unknown (default)
func Triage(
	anomaly domain.CostAnomaly,
	cost CostFetcher,
	infra InfraQuerier,
	kubecost KubeCostQuerier,
	windowStart, windowEnd string,
) (domain.TriageResult, error) {
	if windowStart == "" {
		windowStart = "2026-02-01"
	}
	if windowEnd == "" {
		windowEnd = "2026-02-16"
	}

	ev := domain.TriageEvidence{
		K8sNamespaceDeltas: make(map[string]float64),
	}
	severity := SeverityFromDelta(anomaly.DeltaDollars)
	threshold := 0.2 * math.Max(anomaly.DeltaDollars, 1.0)

	// ---------------------------------------------------------------
	// 1) Commitment coverage drift (RI / SP)
	// ---------------------------------------------------------------
	riCov, err := cost.GetRICoverage(anomaly.AccountID, windowStart, windowEnd)
	if err != nil {
		return domain.TriageResult{}, fmt.Errorf("GetRICoverage: %w", err)
	}
	spCov, err := cost.GetSPCoverage(anomaly.AccountID, windowStart, windowEnd)
	if err != nil {
		return domain.TriageResult{}, fmt.Errorf("GetSPCoverage: %w", err)
	}

	riDelta := floatFromMap(riCov, "coverage_delta", 0.0)
	spDelta := floatFromMap(spCov, "coverage_delta", 0.0)
	ev.RICoverageDelta = float64Ptr(riDelta)
	ev.SPCoverageDelta = float64Ptr(spDelta)

	if math.Abs(riDelta) >= 0.05 || math.Abs(spDelta) >= 0.05 {
		return domain.TriageResult{
			Category:   domain.CategoryCommitmentCoverageDrift,
			Severity:   severity,
			Confidence: 0.8,
			Summary:    "ri/sp coverage shifted materially; investigate commitment coverage/utilization",
			Evidence:   ev,
		}, nil
	}

	// ---------------------------------------------------------------
	// 2) Credits / refunds / fees (CUR line-item types)
	// ---------------------------------------------------------------
	cur, err := cost.GetCURLineItems(anomaly.AccountID, windowStart, windowEnd, anomaly.Service)
	if err != nil {
		return domain.TriageResult{}, fmt.Errorf("GetCURLineItems: %w", err)
	}

	var credits, refunds, fees float64
	for _, item := range cur {
		lineType := strings.ToLower(stringFromMap(item, "line_item_line_item_type", ""))
		amount := floatFromMap(item, "unblended_cost", 0.0)
		switch lineType {
		case "credit":
			credits += amount
		case "refund":
			refunds += amount
		case "fee", "rifee":
			fees += amount
		}
	}
	ev.CreditsDelta = float64Ptr(credits)
	ev.RefundsDelta = float64Ptr(refunds)
	ev.FeesDelta = float64Ptr(fees)

	if math.Abs(credits) >= threshold || math.Abs(refunds) >= threshold {
		return domain.TriageResult{
			Category:   domain.CategoryCreditsRefundsFees,
			Severity:   severity,
			Confidence: 0.75,
			Summary:    "net spend change driven by credits/refunds/fees movement (not usage)",
			Evidence:   ev,
		}, nil
	}

	// ---------------------------------------------------------------
	// 3) Marketplace charges
	// ---------------------------------------------------------------
	var mp float64
	for _, item := range cur {
		productName := strings.ToLower(stringFromMap(item, "product_product_name", ""))
		productCode := strings.ToLower(stringFromMap(item, "line_item_product_code", ""))
		if strings.Contains(productName, "marketplace") || strings.Contains(productCode, "aws marketplace") {
			mp += floatFromMap(item, "unblended_cost", 0.0)
		}
	}
	ev.MarketplaceDelta = float64Ptr(mp)

	if mp >= threshold {
		return domain.TriageResult{
			Category:   domain.CategoryMarketplace,
			Severity:   severity,
			Confidence: 0.8,
			Summary:    "spend appears dominated by marketplace charges (subscription/usage)",
			Evidence:   ev,
		}, nil
	}

	// ---------------------------------------------------------------
	// 4) Data transfer spike
	// ---------------------------------------------------------------
	var dt float64
	for _, item := range cur {
		usageType := strings.ToLower(stringFromMap(item, "line_item_usage_type", ""))
		if strings.Contains(usageType, "datatransfer") {
			dt += floatFromMap(item, "unblended_cost", 0.0)
		}
	}
	ev.DataTransferDelta = float64Ptr(dt)

	if dt >= threshold {
		return domain.TriageResult{
			Category:   domain.CategoryDataTransfer,
			Severity:   severity,
			Confidence: 0.85,
			Summary:    "spike primarily in data transfer usage types",
			Evidence:   ev,
		}, nil
	}

	// ---------------------------------------------------------------
	// 5) KubeCost namespace allocation shift (optional)
	// ---------------------------------------------------------------
	if kubecost != nil {
		alloc, err := kubecost.Allocation("24h", "namespace")
		if err != nil {
			return domain.TriageResult{}, fmt.Errorf("kubecost.Allocation: %w", err)
		}

		allocations, _ := alloc["allocations"].(map[string]any)
		var maxDelta float64
		for ns, raw := range allocations {
			nsMap, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if _, hasDelta := nsMap["delta"]; !hasDelta {
				continue
			}
			delta := floatFromMap(nsMap, "delta", 0.0)
			ev.K8sNamespaceDeltas[ns] = delta
			if delta > maxDelta {
				maxDelta = delta
			}
		}

		if len(ev.K8sNamespaceDeltas) > 0 && maxDelta >= threshold {
			return domain.TriageResult{
				Category:   domain.CategoryK8sCostShift,
				Severity:   severity,
				Confidence: 0.7,
				Summary:    "k8s namespace allocation shifted materially (kubecost)",
				Evidence:   ev,
			}, nil
		}
	}

	// ---------------------------------------------------------------
	// 6) Deploy correlation
	// ---------------------------------------------------------------
	deploys, err := infra.RecentDeploys(anomaly.Service)
	if err != nil {
		return domain.TriageResult{}, fmt.Errorf("RecentDeploys: %w", err)
	}

	if len(deploys) > 0 {
		ids := make([]string, 0, len(deploys))
		for _, d := range deploys {
			id := stringFromMap(d, "id", "deploy")
			ids = append(ids, id)
		}
		ev.DeployCorrelation = ids
		return domain.TriageResult{
			Category:   domain.CategoryDeployRelated,
			Severity:   severity,
			Confidence: 0.7,
			Summary:    "recent deploys detected near anomaly window",
			Evidence:   ev,
		}, nil
	}

	// ---------------------------------------------------------------
	// 7) Expected growth (usage pct vs cost pct)
	// ---------------------------------------------------------------
	metrics, err := infra.CloudWatchMetrics(anomaly.Service, "Requests", "Service")
	if err != nil {
		return domain.TriageResult{}, fmt.Errorf("CloudWatchMetrics: %w", err)
	}

	baseline := floatFromMap(metrics, "baseline", 0.0)
	current := floatFromMap(metrics, "current", 0.0)
	usagePct := PctChange(current, baseline)
	costPct := anomaly.DeltaPercent / 100.0

	if baseline > 0 && usagePct > 0 && math.Abs(usagePct-costPct) <= 0.15 {
		ev.UsageCorrelation = []string{
			fmt.Sprintf("usage pct ~%.2f vs cost pct ~%.2f", usagePct, costPct),
		}
		return domain.TriageResult{
			Category:   domain.CategoryExpectedGrowth,
			Severity:   severity,
			Confidence: 0.8,
			Summary:    "usage increase roughly explains cost increase",
			Evidence:   ev,
		}, nil
	}

	// ---------------------------------------------------------------
	// 8) Unknown (default)
	// ---------------------------------------------------------------
	return domain.TriageResult{
		Category:   domain.CategoryUnknown,
		Severity:   severity,
		Confidence: 0.4,
		Summary:    "no strong deterministic signal; requires deeper analysis",
		Evidence:   ev,
	}, nil
}
