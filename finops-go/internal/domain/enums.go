package domain

import "fmt"

// AnomalySeverity classifies the severity of a cost anomaly.
type AnomalySeverity string

const (
	SeverityLow      AnomalySeverity = "low"
	SeverityMedium   AnomalySeverity = "medium"
	SeverityHigh     AnomalySeverity = "high"
	SeverityCritical AnomalySeverity = "critical"
)

func (s AnomalySeverity) Valid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	}
	return false
}

// AnomalyCategory classifies the root cause category.
type AnomalyCategory string

const (
	CategoryExpectedGrowth          AnomalyCategory = "expected_growth"
	CategoryDeployRelated           AnomalyCategory = "deploy_related"
	CategoryConfigDrift             AnomalyCategory = "config_drift"
	CategoryPricingChange           AnomalyCategory = "pricing_change"
	CategoryCreditsRefundsFees      AnomalyCategory = "credits_refunds_fees"
	CategoryMarketplace             AnomalyCategory = "marketplace"
	CategoryDataTransfer            AnomalyCategory = "data_transfer"
	CategoryK8sCostShift            AnomalyCategory = "k8s_cost_shift"
	CategoryCommitmentCoverageDrift AnomalyCategory = "commitment_coverage_drift"
	CategoryUnknown                 AnomalyCategory = "unknown"
)

func (c AnomalyCategory) Valid() bool {
	switch c {
	case CategoryExpectedGrowth, CategoryDeployRelated, CategoryConfigDrift,
		CategoryPricingChange, CategoryCreditsRefundsFees, CategoryMarketplace,
		CategoryDataTransfer, CategoryK8sCostShift, CategoryCommitmentCoverageDrift,
		CategoryUnknown:
		return true
	}
	return false
}

// ApprovalStatus tracks human-in-the-loop approval state.
type ApprovalStatus string

const (
	ApprovalPending      ApprovalStatus = "pending"
	ApprovalApproved     ApprovalStatus = "approved"
	ApprovalDenied       ApprovalStatus = "denied"
	ApprovalAutoApproved ApprovalStatus = "auto_approved"
	ApprovalTimedOut     ApprovalStatus = "timed_out"
)

func (a ApprovalStatus) Valid() bool {
	switch a {
	case ApprovalPending, ApprovalApproved, ApprovalDenied, ApprovalAutoApproved, ApprovalTimedOut:
		return true
	}
	return false
}

// ActionRiskLevel classifies the risk of a recommended action.
type ActionRiskLevel string

const (
	RiskLow       ActionRiskLevel = "low"
	RiskLowMedium ActionRiskLevel = "low_medium"
	RiskMedium    ActionRiskLevel = "medium"
	RiskHigh      ActionRiskLevel = "high"
	RiskCritical  ActionRiskLevel = "critical"
)

func (r ActionRiskLevel) Valid() bool {
	switch r {
	case RiskLow, RiskLowMedium, RiskMedium, RiskHigh, RiskCritical:
		return true
	}
	return false
}

// RiskScore maps ActionRiskLevel to explicit numeric scores.
// Never rely on enum ordering — use this map.
var RiskScore = map[ActionRiskLevel]int{
	RiskLow:       10,
	RiskLowMedium: 20,
	RiskMedium:    30,
	RiskHigh:      40,
	RiskCritical:  50,
}

// RiskScoreFor returns the numeric risk score, or an error for unknown levels.
func RiskScoreFor(level ActionRiskLevel) (int, error) {
	score, ok := RiskScore[level]
	if !ok {
		return 0, fmt.Errorf("unknown risk level: %q", level)
	}
	return score, nil
}

// VerificationRecommendation is the outcome from the verifier.
type VerificationRecommendation string

const (
	RecommendClose    VerificationRecommendation = "close"
	RecommendRollback VerificationRecommendation = "rollback"
	RecommendEscalate VerificationRecommendation = "escalate"
	RecommendMonitor  VerificationRecommendation = "monitor"
)

func (v VerificationRecommendation) Valid() bool {
	switch v {
	case RecommendClose, RecommendRollback, RecommendEscalate, RecommendMonitor:
		return true
	}
	return false
}
