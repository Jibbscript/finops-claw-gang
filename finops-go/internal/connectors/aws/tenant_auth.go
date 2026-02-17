package aws

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var roleARNRe = regexp.MustCompile(`^arn:aws:iam::\d{12}:role/.+$`)

// ValidateRoleARN checks that the ARN looks like a valid IAM role ARN.
func ValidateRoleARN(arn string) error {
	if !roleARNRe.MatchString(arn) {
		return fmt.Errorf("invalid IAM role ARN: %q", arn)
	}
	return nil
}

type cachedConfig struct {
	cfg       aws.Config
	expiresAt time.Time
}

// TenantConfigProvider caches per-tenant assumed-role AWS configs.
// Sessions are refreshed 5 minutes before STS expiry.
type TenantConfigProvider struct {
	baseRegion  string
	baseProfile string

	mu    sync.RWMutex
	cache map[string]*cachedConfig

	// sessionDuration is the STS session length (default 1h).
	sessionDuration time.Duration
	// refreshBefore is how far ahead of expiry to refresh (default 5m).
	refreshBefore time.Duration

	// now is injectable for testing.
	now func() time.Time
}

// NewTenantConfigProvider creates a provider with the given base AWS config.
func NewTenantConfigProvider(region, profile string) *TenantConfigProvider {
	return &TenantConfigProvider{
		baseRegion:      region,
		baseProfile:     profile,
		cache:           make(map[string]*cachedConfig),
		sessionDuration: time.Hour,
		refreshBefore:   5 * time.Minute,
		now:             time.Now,
	}
}

// cacheKey returns a unique key for a (tenantID, roleARN) pair.
func cacheKey(tenantID, roleARN string) string {
	return tenantID + "|" + roleARN
}

// ForTenant returns an AWS config with credentials assumed from the tenant's role.
// Results are cached and automatically refreshed before expiry.
func (p *TenantConfigProvider) ForTenant(ctx context.Context, tenantID, roleARN, region string) (aws.Config, error) {
	if err := ValidateRoleARN(roleARN); err != nil {
		return aws.Config{}, err
	}

	key := cacheKey(tenantID, roleARN)

	// Fast path: check cache under read lock.
	p.mu.RLock()
	if cached, ok := p.cache[key]; ok && p.now().Before(cached.expiresAt.Add(-p.refreshBefore)) {
		cfg := cached.cfg
		p.mu.RUnlock()
		if region != "" {
			cfg.Region = region
		}
		return cfg, nil
	}
	p.mu.RUnlock()

	// Slow path: create new assumed-role config.
	r := p.baseRegion
	if region != "" {
		r = region
	}

	cfg, err := p.assumeRole(ctx, roleARN, tenantID, r)
	if err != nil {
		return aws.Config{}, fmt.Errorf("assume role for tenant %s: %w", tenantID, err)
	}

	p.mu.Lock()
	p.cache[key] = &cachedConfig{
		cfg:       cfg,
		expiresAt: p.now().Add(p.sessionDuration),
	}
	p.mu.Unlock()

	return cfg, nil
}

func (p *TenantConfigProvider) assumeRole(ctx context.Context, roleARN, sessionSuffix, region string) (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	if p.baseProfile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(p.baseProfile))
	}

	baseCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load base config: %w", err)
	}

	stsClient := sts.NewFromConfig(baseCfg)
	baseCfg.Credentials = stscreds.NewAssumeRoleProvider(stsClient, roleARN,
		func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = "finops-" + sessionSuffix
			o.Duration = p.sessionDuration
		},
	)

	return baseCfg, nil
}
