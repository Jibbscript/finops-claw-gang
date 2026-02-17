package aws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRoleARN(t *testing.T) {
	t.Parallel()
	tests := []struct {
		arn     string
		wantErr bool
	}{
		{"arn:aws:iam::123456789012:role/MyRole", false},
		{"arn:aws:iam::123456789012:role/path/MyRole", false},
		{"arn:aws:iam::12345:role/Short", true},           // too few digits
		{"arn:aws:iam::123456789012:user/NotARole", true}, // user, not role
		{"", true},
		{"not-an-arn", true},
	}

	for _, tt := range tests {
		t.Run(tt.arn, func(t *testing.T) {
			t.Parallel()
			err := ValidateRoleARN(tt.arn)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTenantConfigProvider_CacheHit(t *testing.T) {
	p := NewTenantConfigProvider("us-east-1", "")

	now := time.Now()
	p.now = func() time.Time { return now }

	// Pre-populate cache.
	key := cacheKey("tenant-1", "arn:aws:iam::123456789012:role/Test")
	p.cache[key] = &cachedConfig{
		expiresAt: now.Add(time.Hour),
	}

	cfg, err := p.ForTenant(t.Context(), "tenant-1", "arn:aws:iam::123456789012:role/Test", "eu-west-1")
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", cfg.Region)
}

func TestTenantConfigProvider_InvalidARN(t *testing.T) {
	t.Parallel()
	p := NewTenantConfigProvider("us-east-1", "")
	_, err := p.ForTenant(t.Context(), "tenant-1", "invalid-arn", "")
	assert.Error(t, err)
}

func TestTenantConfigProvider_CacheRefreshed(t *testing.T) {
	p := NewTenantConfigProvider("us-east-1", "")

	now := time.Now()
	p.now = func() time.Time { return now }

	arn := "arn:aws:iam::123456789012:role/Test"
	key := cacheKey("tenant-1", arn)

	// Pre-populate with entry inside refresh window (needs refresh).
	oldExpiry := now.Add(2 * time.Minute) // less than refreshBefore (5m)
	p.cache[key] = &cachedConfig{
		expiresAt: oldExpiry,
	}

	// ForTenant will detect cache miss (within refresh window) and
	// re-assume the role. Since credentials are lazy, this succeeds.
	_, err := p.ForTenant(t.Context(), "tenant-1", arn, "")
	require.NoError(t, err)

	// Verify the cache entry was refreshed with a new expiry.
	p.mu.RLock()
	cached := p.cache[key]
	p.mu.RUnlock()
	require.NotNil(t, cached)
	assert.True(t, cached.expiresAt.After(oldExpiry), "cache should have new expiry")
}
