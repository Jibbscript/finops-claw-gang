// Package connectors provides composite adapters that compose individual AWS
// service clients to satisfy the union interfaces consumed by activities.
package connectors

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/athena"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/cloudwatch"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/codedeploy"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/costexplorer"
	"github.com/finops-claw-gang/finops-go/internal/connectors/aws/tagging"
	"github.com/finops-claw-gang/finops-go/internal/ratelimit"
)

// rateLimitTimeout is a safety net for rate limiter waits.
// The interface methods don't carry context, so we use a fixed timeout
// to prevent indefinite blocking if the limiter is severely constrained.
const rateLimitTimeout = 30 * time.Second

// AWSCostClient satisfies activities.CostDeps by composing Cost Explorer and Athena clients.
type AWSCostClient struct {
	ce      *costexplorer.Client
	ath     *athena.Querier
	limiter *ratelimit.ServiceLimiter // nil = no limiting
}

// NewAWSCostClient creates an AWSCostClient from an AWS config and Athena CUR configuration.
func NewAWSCostClient(cfg aws.Config, curDatabase, curTable, curWorkgroup, curOutputBucket string) *AWSCostClient {
	return &AWSCostClient{
		ce:  costexplorer.New(cfg),
		ath: athena.New(cfg, curDatabase, curTable, curWorkgroup, curOutputBucket),
	}
}

// SetLimiter attaches a rate limiter to the client.
func (c *AWSCostClient) SetLimiter(sl *ratelimit.ServiceLimiter) {
	c.limiter = sl
}

func (c *AWSCostClient) waitCE() error {
	if c.limiter != nil {
		ctx, cancel := context.WithTimeout(context.Background(), rateLimitTimeout)
		defer cancel()
		return c.limiter.Wait(ctx, "CostExplorer")
	}
	return nil
}

func (c *AWSCostClient) waitAthena() error {
	if c.limiter != nil {
		ctx, cancel := context.WithTimeout(context.Background(), rateLimitTimeout)
		defer cancel()
		return c.limiter.Wait(ctx, "Athena")
	}
	return nil
}

func (c *AWSCostClient) GetRICoverage(accountID, startDate, endDate string) (map[string]any, error) {
	if err := c.waitCE(); err != nil {
		return nil, err
	}
	return c.ce.GetRICoverage(accountID, startDate, endDate)
}

func (c *AWSCostClient) GetSPCoverage(accountID, startDate, endDate string) (map[string]any, error) {
	if err := c.waitCE(); err != nil {
		return nil, err
	}
	return c.ce.GetSPCoverage(accountID, startDate, endDate)
}

func (c *AWSCostClient) GetCostTimeseries(service, accountID, startDate, endDate string) (map[string]any, error) {
	if err := c.waitCE(); err != nil {
		return nil, err
	}
	return c.ce.GetCostTimeseries(service, accountID, startDate, endDate)
}

func (c *AWSCostClient) GetCURLineItems(accountID, startDate, endDate, service string) ([]map[string]any, error) {
	if err := c.waitAthena(); err != nil {
		return nil, err
	}
	return c.ath.GetCURLineItems(accountID, startDate, endDate, service)
}

// AWSInfraClient satisfies activities.InfraDeps by composing CloudWatch, Tagging, and CodeDeploy clients.
type AWSInfraClient struct {
	cw      *cloudwatch.Client
	tg      *tagging.Client
	cd      *codedeploy.Client
	limiter *ratelimit.ServiceLimiter // nil = no limiting
}

// NewAWSInfraClient creates an AWSInfraClient from an AWS config.
func NewAWSInfraClient(cfg aws.Config) *AWSInfraClient {
	return &AWSInfraClient{
		cw: cloudwatch.New(cfg),
		tg: tagging.New(cfg),
		cd: codedeploy.New(cfg),
	}
}

// SetLimiter attaches a rate limiter to the client.
func (c *AWSInfraClient) SetLimiter(sl *ratelimit.ServiceLimiter) {
	c.limiter = sl
}

func (c *AWSInfraClient) RecentDeploys(service string) ([]map[string]any, error) {
	return c.cd.RecentDeploys(service)
}

func (c *AWSInfraClient) CloudWatchMetrics(resourceID, metricName, namespace string) (map[string]any, error) {
	if c.limiter != nil {
		ctx, cancel := context.WithTimeout(context.Background(), rateLimitTimeout)
		defer cancel()
		if err := c.limiter.Wait(ctx, "CloudWatch"); err != nil {
			return nil, err
		}
	}
	return c.cw.CloudWatchMetrics(resourceID, metricName, namespace)
}

func (c *AWSInfraClient) ResourceTags(resourceARN string) (map[string]string, error) {
	return c.tg.ResourceTags(resourceARN)
}
