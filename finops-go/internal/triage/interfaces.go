package triage

import (
	"context"

	"github.com/finops-claw-gang/finops-go/internal/domain"
)

// CostFetcher provides cost data needed by the triage classifier.
type CostFetcher interface {
	GetRICoverage(accountID, startDate, endDate string) (map[string]any, error)
	GetSPCoverage(accountID, startDate, endDate string) (map[string]any, error)
	GetCURLineItems(accountID, startDate, endDate string, service string) ([]map[string]any, error)
}

// InfraQuerier provides infrastructure data needed by the triage classifier.
type InfraQuerier interface {
	RecentDeploys(service string) ([]map[string]any, error)
	CloudWatchMetrics(resourceID, metricName, namespace string) (map[string]any, error)
}

// KubeCostQuerier provides KubeCost allocation data.
type KubeCostQuerier interface {
	Allocation(window, aggregate string) (map[string]any, error)
}

// WasteQuerier provides resource waste data from aws-doctor scans.
type WasteQuerier interface {
	Waste(ctx context.Context, accountID, region string) ([]domain.WasteFinding, error)
}
