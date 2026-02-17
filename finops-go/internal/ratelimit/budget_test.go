package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityBudget_UnderLimit(t *testing.T) {
	b := NewActivityBudget(5, time.Minute)

	err := b.Check("tenant-1", "TriageAnomaly")
	require.NoError(t, err)

	b.Record("tenant-1", "TriageAnomaly")
	b.Record("tenant-1", "TriageAnomaly")

	err = b.Check("tenant-1", "TriageAnomaly")
	assert.NoError(t, err)
}

func TestActivityBudget_ExceedsLimit(t *testing.T) {
	b := NewActivityBudget(2, time.Minute)

	b.Record("tenant-1", "TriageAnomaly")
	b.Record("tenant-1", "TriageAnomaly")

	err := b.Check("tenant-1", "TriageAnomaly")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "budget exceeded")
}

func TestActivityBudget_WindowReset(t *testing.T) {
	b := NewActivityBudget(2, time.Minute)

	now := time.Now()
	b.now = func() time.Time { return now }

	b.Record("tenant-1", "TriageAnomaly")
	b.Record("tenant-1", "TriageAnomaly")
	err := b.Check("tenant-1", "TriageAnomaly")
	assert.Error(t, err)

	// Advance time past window.
	b.now = func() time.Time { return now.Add(2 * time.Minute) }
	err = b.Check("tenant-1", "TriageAnomaly")
	assert.NoError(t, err)
}

func TestActivityBudget_DifferentTenants(t *testing.T) {
	b := NewActivityBudget(1, time.Minute)

	b.Record("tenant-1", "TriageAnomaly")
	err := b.Check("tenant-1", "TriageAnomaly")
	assert.Error(t, err)

	// Different tenant should have its own budget.
	err = b.Check("tenant-2", "TriageAnomaly")
	assert.NoError(t, err)
}
