package connectors

import (
	"context"

	"github.com/finops-claw-gang/finops-go/internal/connectors/aws"
	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/temporal/activities"
)

// TenantClientFactory creates per-tenant AWS clients using assumed-role sessions.
// It implements activities.TenantDeps.
type TenantClientFactory struct {
	provider        *aws.TenantConfigProvider
	curDatabase     string
	curTable        string
	curWorkgroup    string
	curOutputBucket string
}

// Compile-time check.
var _ activities.TenantDeps = (*TenantClientFactory)(nil)

// NewTenantClientFactory creates a factory backed by the given config provider.
func NewTenantClientFactory(
	provider *aws.TenantConfigProvider,
	curDatabase, curTable, curWorkgroup, curOutputBucket string,
) *TenantClientFactory {
	return &TenantClientFactory{
		provider:        provider,
		curDatabase:     curDatabase,
		curTable:        curTable,
		curWorkgroup:    curWorkgroup,
		curOutputBucket: curOutputBucket,
	}
}

// CostClient creates a per-tenant AWSCostClient.
func (f *TenantClientFactory) CostClient(ctx context.Context, tenant domain.TenantContext) (activities.CostDeps, error) {
	cfg, err := f.provider.ForTenant(ctx, tenant.TenantID, tenant.IAMRoleARN, tenant.DefaultRegion)
	if err != nil {
		return nil, err
	}
	return NewAWSCostClient(cfg, f.curDatabase, f.curTable, f.curWorkgroup, f.curOutputBucket), nil
}

// InfraClient creates a per-tenant AWSInfraClient.
func (f *TenantClientFactory) InfraClient(ctx context.Context, tenant domain.TenantContext) (activities.InfraDeps, error) {
	cfg, err := f.provider.ForTenant(ctx, tenant.TenantID, tenant.IAMRoleARN, tenant.DefaultRegion)
	if err != nil {
		return nil, err
	}
	return NewAWSInfraClient(cfg), nil
}
