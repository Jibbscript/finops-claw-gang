package ratelimit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceLimiter_Wait(t *testing.T) {
	sl := NewServiceLimiter(ServiceRates{CostExplorer: 100, Athena: 100, CloudWatch: 100, STS: 100})

	// Should not block at high rate.
	err := sl.Wait(context.Background(), "CostExplorer")
	require.NoError(t, err)
}

func TestServiceLimiter_UnknownService(t *testing.T) {
	sl := NewServiceLimiter(DefaultServiceRates())

	// Unknown service should pass through.
	err := sl.Wait(context.Background(), "UnknownService")
	assert.NoError(t, err)
}

func TestServiceLimiter_CancelledContext(t *testing.T) {
	// Create a very restrictive limiter.
	sl := NewServiceLimiter(ServiceRates{CostExplorer: 0.001})

	// Consume the burst.
	_ = sl.Wait(context.Background(), "CostExplorer")

	// Next call with cancelled context should error.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sl.Wait(ctx, "CostExplorer")
	assert.Error(t, err)
}
