// Package ratelimit provides token-bucket rate limiters and per-tenant activity budgets.
package ratelimit

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/time/rate"
)

// ServiceRates configures per-service request rates (requests per second).
type ServiceRates struct {
	CostExplorer float64
	Athena       float64
	CloudWatch   float64
	STS          float64
}

// DefaultServiceRates returns conservative AWS rate limits.
func DefaultServiceRates() ServiceRates {
	return ServiceRates{
		CostExplorer: 5,
		Athena:       5,
		CloudWatch:   20,
		STS:          10,
	}
}

// ServiceLimiter rate-limits AWS API calls per service using token buckets.
type ServiceLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

// NewServiceLimiter creates a limiter with the given per-service rates.
func NewServiceLimiter(rates ServiceRates) *ServiceLimiter {
	limiters := map[string]*rate.Limiter{
		"CostExplorer": rate.NewLimiter(rate.Limit(rates.CostExplorer), int(rates.CostExplorer)),
		"Athena":       rate.NewLimiter(rate.Limit(rates.Athena), int(rates.Athena)),
		"CloudWatch":   rate.NewLimiter(rate.Limit(rates.CloudWatch), int(rates.CloudWatch)),
		"STS":          rate.NewLimiter(rate.Limit(rates.STS), int(rates.STS)),
	}
	return &ServiceLimiter{limiters: limiters}
}

// Wait blocks until a token is available for the named service, or ctx is cancelled.
func (sl *ServiceLimiter) Wait(ctx context.Context, service string) error {
	sl.mu.RLock()
	limiter, ok := sl.limiters[service]
	sl.mu.RUnlock()
	if !ok {
		return nil // unknown service = no limit
	}
	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit %s: %w", service, err)
	}
	return nil
}
