package ratelimit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceLimiter_Wait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rates   ServiceRates
		service string
		wantErr bool
	}{
		{"passes at high rate", ServiceRates{CostExplorer: 100, Athena: 100, CloudWatch: 100, STS: 100}, "CostExplorer", false},
		{"unknown service passes through", DefaultServiceRates(), "UnknownService", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sl := NewServiceLimiter(tt.rates)
			err := sl.Wait(context.Background(), tt.service)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("cancelled context errors", func(t *testing.T) {
		t.Parallel()
		sl := NewServiceLimiter(ServiceRates{CostExplorer: 0.001})
		_ = sl.Wait(context.Background(), "CostExplorer") // consume burst
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := sl.Wait(ctx, "CostExplorer")
		assert.Error(t, err)
	})
}
